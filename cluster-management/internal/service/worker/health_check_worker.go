package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
)

type HealthCheckWorker struct {
	clusterRepo     *repository.ClusterRepository
	stateRepo       *repository.ClusterStateRepository
	clusterManager  *service.ClusterManager
	encryptionSvc   *service.EncryptionService
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
	checkInterval   time.Duration
	maxConcurrency  int
	sem             chan struct{}
}

func NewHealthCheckWorker(
	clusterRepo *repository.ClusterRepository,
	stateRepo *repository.ClusterStateRepository,
	clusterManager *service.ClusterManager,
	encryptionSvc *service.EncryptionService,
) *HealthCheckWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &HealthCheckWorker{
		clusterRepo:     clusterRepo,
		stateRepo:       stateRepo,
		clusterManager:  clusterManager,
		encryptionSvc:   encryptionSvc,
		ctx:             ctx,
		cancel:          cancel,
		checkInterval:   5 * time.Minute,
		maxConcurrency:  10,
		sem:             make(chan struct{}, 10),
	}
}

func (w *HealthCheckWorker) Start() {
	log.Println("Starting health check worker...")

	w.performHealthCheck()

	w.wg.Add(1)
	go w.scheduler()
}

func (w *HealthCheckWorker) Stop() {
	log.Println("Stopping health check worker...")
	w.cancel()
	w.wg.Wait()
	close(w.sem)
}

func (w *HealthCheckWorker) scheduler() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.performHealthCheck()
		}
	}
}

func (w *HealthCheckWorker) performHealthCheck() {
	log.Println("Starting scheduled health check...")

	clusters, err := w.clusterRepo.FindActiveClusters()
	if err != nil {
		log.Printf("Failed to fetch clusters: %v", err)
		return
	}

	log.Printf("Found %d active clusters", len(clusters))

	for _, cluster := range clusters {
		w.wg.Add(1)
		go func(c model.Cluster) {
			defer w.wg.Done()
			w.checkCluster(c)
		}(*cluster)
	}
}

func (w *HealthCheckWorker) checkCluster(cluster model.Cluster) {
	w.sem <- struct{}{}
	defer func() { <-w.sem }()

	kubeconfig, err := w.encryptionSvc.Decrypt(
		cluster.KubeconfigEncrypted,
		cluster.KubeconfigNonce,
	)
	if err != nil {
		log.Printf("Failed to decrypt kubeconfig for cluster %s: %v", cluster.Name, err)
		w.updateClusterState(cluster.ID, false, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	clientset, err := w.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		log.Printf("Failed to get client for cluster %s: %v", cluster.Name, err)
		w.updateClusterState(cluster.ID, false, err.Error())
		return
	}

	result, err := w.clusterManager.HealthCheck(ctx, clientset)
	if err != nil {
		log.Printf("Health check failed for cluster %s: %v", cluster.Name, err)
		w.updateClusterState(cluster.ID, false, err.Error())
		return
	}

	w.updateClusterState(cluster.ID, true, result)
	log.Printf("Health check completed for cluster %s: %s", cluster.Name, result.Status)
}

func (w *HealthCheckWorker) updateClusterState(
	clusterID uuid.UUID,
	success bool,
	result interface{},
) {
	state := &model.ClusterState{
		ClusterID: clusterID,
		SyncSuccess: success,
		LastSyncAt: time.Now(),
	}

	if !success {
		state.Status = "disconnected"
		if errMsg, ok := result.(string); ok {
			state.SyncError = errMsg
		}
	} else if result != nil {
		if healthResult, ok := result.(*service.HealthCheckResult); ok {
			state.Status = healthResult.Status
			state.NodeCount = healthResult.NodeCount
			state.TotalCPUCores = healthResult.TotalCPUCores
			state.TotalMemoryBytes = healthResult.TotalMemoryBytes
			state.KubernetesVersion = healthResult.Version
			state.APIServerURL = healthResult.APIServerURL
			state.LastHeartbeatAt = &healthResult.LastHeartbeatAt
		}
	}

	if err := w.stateRepo.Upsert(state); err != nil {
		log.Printf("Failed to update cluster state: %v", err)
	}
}

func (w *HealthCheckWorker) TriggerSync(clusterID uuid.UUID) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		cluster, err := w.clusterRepo.GetByID(clusterID.String())
		if err != nil {
			log.Printf("Failed to get cluster for sync: %v", err)
			return
		}
		w.checkCluster(*cluster)
	}()
}
