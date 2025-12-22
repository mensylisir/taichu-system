package worker

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
)

// ResourceCache 资源缓存接口
type ResourceCache interface {
	GetNodes(clusterID uuid.UUID) ([]model.Node, bool)
	SetNodes(clusterID uuid.UUID, nodes []model.Node)
	GetEvents(clusterID uuid.UUID) ([]model.Event, bool)
	SetEvents(clusterID uuid.UUID, events []model.Event)
	GetClusterResources(clusterID uuid.UUID) (*model.ClusterResource, bool)
	SetClusterResources(clusterID uuid.UUID, resources *model.ClusterResource)
	InvalidateCluster(clusterID uuid.UUID)
}

// MemoryResourceCache 内存中的资源缓存实现
type MemoryResourceCache struct {
	nodeCache     map[uuid.UUID][]model.Node
	eventCache    map[uuid.UUID][]model.Event
	resourceCache map[uuid.UUID]*model.ClusterResource
	cacheTTL      time.Duration
	lastUpdated   map[uuid.UUID]time.Time
	mutex         sync.RWMutex
}

// NewMemoryResourceCache 创建一个新的内存资源缓存
func NewMemoryResourceCache(ttl time.Duration) *MemoryResourceCache {
	return &MemoryResourceCache{
		nodeCache:     make(map[uuid.UUID][]model.Node),
		eventCache:    make(map[uuid.UUID][]model.Event),
		resourceCache: make(map[uuid.UUID]*model.ClusterResource),
		cacheTTL:      ttl,
		lastUpdated:   make(map[uuid.UUID]time.Time),
	}
}

// GetNodes 从缓存中获取节点信息
func (c *MemoryResourceCache) GetNodes(clusterID uuid.UUID) ([]model.Node, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.isCacheValid(clusterID) {
		return nil, false
	}

	nodes, exists := c.nodeCache[clusterID]
	if !exists {
		return nil, false
	}

	// 返回副本以避免并发修改
	result := make([]model.Node, len(nodes))
	copy(result, nodes)
	return result, true
}

// SetNodes 将节点信息存储到缓存中
func (c *MemoryResourceCache) SetNodes(clusterID uuid.UUID, nodes []model.Node) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 存储副本以避免并发修改
	storedNodes := make([]model.Node, len(nodes))
	copy(storedNodes, nodes)
	c.nodeCache[clusterID] = storedNodes
	c.lastUpdated[clusterID] = time.Now()
}

// GetEvents 从缓存中获取事件信息
func (c *MemoryResourceCache) GetEvents(clusterID uuid.UUID) ([]model.Event, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.isCacheValid(clusterID) {
		return nil, false
	}

	events, exists := c.eventCache[clusterID]
	if !exists {
		return nil, false
	}

	// 返回副本以避免并发修改
	result := make([]model.Event, len(events))
	copy(result, events)
	return result, true
}

// SetEvents 将事件信息存储到缓存中
func (c *MemoryResourceCache) SetEvents(clusterID uuid.UUID, events []model.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 存储副本以避免并发修改
	storedEvents := make([]model.Event, len(events))
	copy(storedEvents, events)
	c.eventCache[clusterID] = storedEvents
	c.lastUpdated[clusterID] = time.Now()
}

// GetClusterResources 从缓存中获取集群资源信息
func (c *MemoryResourceCache) GetClusterResources(clusterID uuid.UUID) (*model.ClusterResource, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.isCacheValid(clusterID) {
		return nil, false
	}

	resources, exists := c.resourceCache[clusterID]
	if !exists {
		return nil, false
	}

	// 返回副本以避免并发修改
	result := &model.ClusterResource{
		ClusterID:         resources.ClusterID,
		TotalCPUCores:     resources.TotalCPUCores,
		TotalMemoryBytes:  resources.TotalMemoryBytes,
		TotalStorageBytes: resources.TotalStorageBytes,
		UsedCPUCores:      resources.UsedCPUCores,
		UsedMemoryBytes:   resources.UsedMemoryBytes,
		UsedStorageBytes:  resources.UsedStorageBytes,
		// PodCount, NodeCount, NamespaceCount 字段在 ClusterResource 模型中不存在
	}
	return result, true
}

// SetClusterResources 将集群资源信息存储到缓存中
func (c *MemoryResourceCache) SetClusterResources(clusterID uuid.UUID, resources *model.ClusterResource) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 存储副本以避免并发修改
	c.resourceCache[clusterID] = &model.ClusterResource{
		ClusterID:         resources.ClusterID,
		TotalCPUCores:     resources.TotalCPUCores,
		TotalMemoryBytes:  resources.TotalMemoryBytes,
		TotalStorageBytes: resources.TotalStorageBytes,
		UsedCPUCores:      resources.UsedCPUCores,
		UsedMemoryBytes:   resources.UsedMemoryBytes,
		UsedStorageBytes:  resources.UsedStorageBytes,
		// PodCount, NodeCount, NamespaceCount 字段在 ClusterResource 模型中不存在
	}
	c.lastUpdated[clusterID] = time.Now()
}

// InvalidateCluster 使指定集群的缓存失效
func (c *MemoryResourceCache) InvalidateCluster(clusterID uuid.UUID) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.nodeCache, clusterID)
	delete(c.eventCache, clusterID)
	delete(c.resourceCache, clusterID)
	delete(c.lastUpdated, clusterID)
}

// isCacheValid 检查缓存是否仍然有效
func (c *MemoryResourceCache) isCacheValid(clusterID uuid.UUID) bool {
	lastUpdate, exists := c.lastUpdated[clusterID]
	if !exists {
		return false
	}

	return time.Since(lastUpdate) < c.cacheTTL
}

// CleanupExpired 清理过期的缓存条目
func (c *MemoryResourceCache) CleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for clusterID, lastUpdate := range c.lastUpdated {
		if time.Since(lastUpdate) > c.cacheTTL {
			delete(c.nodeCache, clusterID)
			delete(c.eventCache, clusterID)
			delete(c.resourceCache, clusterID)
			delete(c.lastUpdated, clusterID)
		}
	}
}