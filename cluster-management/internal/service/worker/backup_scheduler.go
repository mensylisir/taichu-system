package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
)

type BackupScheduler struct {
	backupRepo         *repository.BackupRepository
	backupScheduleRepo *repository.BackupScheduleRepository
	clusterRepo        *repository.ClusterRepository
	backupService      *service.BackupService
	cron               *cron.Cron
	wg                 sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
	runningBackups     sync.Map
}

func NewBackupScheduler(
	backupRepo *repository.BackupRepository,
	backupScheduleRepo *repository.BackupScheduleRepository,
	clusterRepo *repository.ClusterRepository,
	backupService *service.BackupService,
) *BackupScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &BackupScheduler{
		backupRepo:         backupRepo,
		backupScheduleRepo: backupScheduleRepo,
		clusterRepo:        clusterRepo,
		backupService:      backupService,
		cron:               cron.New(cron.WithSeconds()),
		ctx:                ctx,
		cancel:             cancel,
	}
}

func (s *BackupScheduler) Start() {
	log.Println("Starting backup scheduler...")

	// 加载所有启用的定时任务
	s.loadSchedules()

	// 启动cron调度器
	s.cron.Start()

	// 定期检查新添加的定时任务
	s.wg.Add(1)
	go s.scheduleWatcher()

	log.Println("Backup scheduler started successfully")
}

func (s *BackupScheduler) Stop() {
	log.Println("Stopping backup scheduler...")
	s.cancel()
	s.cron.Stop()

	s.runningBackups.Range(func(key, value interface{}) bool {
		backupID := key.(string)
		log.Printf("Waiting for backup %s to complete...", backupID)
		return true
	})

	s.wg.Wait()
	log.Println("Backup scheduler stopped")
}

func (s *BackupScheduler) loadSchedules() {
	// 获取所有启用的定时任务
	schedules, err := s.backupScheduleRepo.ListEnabled()
	if err != nil {
		log.Printf("Failed to load backup schedules: %v", err)
		return
	}

	for _, schedule := range schedules {
		s.addSchedule(schedule)
	}

	log.Printf("Loaded %d backup schedules", len(schedules))
}

func (s *BackupScheduler) addSchedule(schedule *model.BackupSchedule) {
	// 添加cron任务
	entryID, err := s.cron.AddFunc(schedule.CronExpr, func() {
		s.executeBackupSchedule(schedule)
	})
	if err != nil {
		log.Printf("Failed to add schedule %s: %v", schedule.ID, err)
		return
	}

	log.Printf("Added backup schedule %s with cron %s (entry ID: %v)", schedule.Name, schedule.CronExpr, entryID)
}

func (s *BackupScheduler) executeBackupSchedule(schedule *model.BackupSchedule) {
	log.Printf("Executing backup schedule: %s", schedule.Name)

	now := time.Now()
	schedule.LastRunAt = &now
	if err := s.backupScheduleRepo.Update(schedule); err != nil {
		log.Printf("Failed to update last run time: %v", err)
	}

	backupName := fmt.Sprintf("%s-%s", schedule.Name, now.Format("20060102-150405"))
	backup, err := s.backupService.CreateBackup(
		schedule.ClusterID.String(),
		backupName,
		schedule.BackupType,
		schedule.RetentionDays,
	)
	if err != nil {
		log.Printf("Failed to create backup for schedule %s: %v", schedule.Name, err)
		return
	}

	s.runningBackups.Store(backup.ID.String(), struct{}{})
	go func() {
		defer s.runningBackups.Delete(backup.ID.String())

		if err := s.backupService.ExecuteBackup(backup.ID.String()); err != nil {
			log.Printf("Failed to execute backup %s for schedule %s: %v", backup.ID, schedule.Name, err)
		} else {
			log.Printf("Backup %s for schedule %s completed successfully", backup.ID, schedule.Name)
		}
	}()

	log.Printf("Created backup %s for schedule %s", backup.ID, schedule.Name)
}

func (s *BackupScheduler) scheduleWatcher() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.loadSchedules()
		}
	}
}

// ReloadSchedules 重新加载所有定时任务
func (s *BackupScheduler) ReloadSchedules() {
	log.Println("Reloading backup schedules...")
	s.loadSchedules()
	log.Println("Backup schedules reloaded")
}
