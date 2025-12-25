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
	alertService       *service.AlertService
	cron               *cron.Cron
	wg                 sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
	runningBackups     sync.Map
	runningSchedules   sync.Map
	mu                 sync.Mutex
	scheduleEntries    sync.Map
}

func NewBackupScheduler(
	backupRepo *repository.BackupRepository,
	backupScheduleRepo *repository.BackupScheduleRepository,
	clusterRepo *repository.ClusterRepository,
	backupService *service.BackupService,
	alertService *service.AlertService,
) *BackupScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &BackupScheduler{
		backupRepo:         backupRepo,
		backupScheduleRepo: backupScheduleRepo,
		clusterRepo:        clusterRepo,
		backupService:      backupService,
		alertService:       alertService,
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
	s.mu.Lock()
	defer s.mu.Unlock()

	schedules, err := s.backupScheduleRepo.ListEnabled()
	if err != nil {
		log.Printf("Failed to load backup schedules: %v", err)
		return
	}

	loadedScheduleIDs := make(map[string]bool)
	for _, schedule := range schedules {
		loadedScheduleIDs[schedule.ID.String()] = true
		s.addSchedule(schedule)
	}

	s.scheduleEntries.Range(func(key, value interface{}) bool {
		scheduleID := key.(string)
		if !loadedScheduleIDs[scheduleID] {
			if entryID, ok := value.(cron.EntryID); ok {
				s.cron.Remove(entryID)
				s.scheduleEntries.Delete(scheduleID)
				log.Printf("Removed disabled backup schedule %s", scheduleID)
			}
		}
		return true
	})

	log.Printf("Loaded %d backup schedules", len(schedules))
}

func (s *BackupScheduler) addSchedule(schedule *model.BackupSchedule) {
	scheduleID := schedule.ID.String()

	if _, exists := s.scheduleEntries.Load(scheduleID); exists {
		log.Printf("Backup schedule %s already exists, skipping", schedule.Name)
		return
	}

	entryID, err := s.cron.AddFunc(schedule.CronExpr, func() {
		s.executeBackupSchedule(schedule)
	})
	if err != nil {
		log.Printf("Failed to add schedule %s: %v", schedule.ID, err)
		return
	}

	s.scheduleEntries.Store(scheduleID, entryID)
	log.Printf("Added backup schedule %s with cron %s (entry ID: %v)", schedule.Name, schedule.CronExpr, entryID)
}

func (s *BackupScheduler) executeBackupSchedule(schedule *model.BackupSchedule) {
	log.Printf("Executing backup schedule: %s", schedule.Name)

	scheduleID := schedule.ID.String()

	if _, running := s.runningSchedules.LoadOrStore(scheduleID, struct{}{}); running {
		log.Printf("Backup schedule %s is already running, skipping execution", schedule.Name)
		return
	}
	defer s.runningSchedules.Delete(scheduleID)

	now := time.Now()
	schedule.LastRunAt = &now
	if err := s.backupScheduleRepo.Update(schedule); err != nil {
		log.Printf("Failed to update last run time: %v", err)
		if s.alertService != nil {
			s.alertService.AlertScheduleFailed(scheduleID, schedule.ClusterID.String(), err.Error())
		}
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
		if s.alertService != nil {
			s.alertService.AlertScheduleFailed(scheduleID, schedule.ClusterID.String(), err.Error())
		}
		return
	}

	s.runningBackups.Store(backup.ID.String(), struct{}{})
	go func() {
		defer s.runningBackups.Delete(backup.ID.String())

		if err := s.backupService.ExecuteBackup(backup.ID.String()); err != nil {
			log.Printf("Failed to execute backup %s for schedule %s: %v", backup.ID, schedule.Name, err)
			if s.alertService != nil {
				s.alertService.AlertScheduleFailed(scheduleID, schedule.ClusterID.String(), err.Error())
			}
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
