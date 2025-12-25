package service

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LimitRangeManager LimitRange管理器
type LimitRangeManager struct {
	kubernetesClient *kubernetes.Clientset
}

// NewLimitRangeManager 创建LimitRange管理器
func NewLimitRangeManager(kubeClient *kubernetes.Clientset) *LimitRangeManager {
	return &LimitRangeManager{
		kubernetesClient: kubeClient,
	}
}

// CreateDefaultLimitRange 为命名空间创建默认的LimitRange
func (l *LimitRangeManager) CreateDefaultLimitRange(namespace string, cpuRequest, cpuLimit, memoryRequest, memoryLimit string) error {
	ctx := context.Background()

	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-limit-range",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    l.parseQuantity(cpuLimit),
						corev1.ResourceMemory: l.parseQuantity(memoryLimit),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    l.parseQuantity(cpuRequest),
						corev1.ResourceMemory: l.parseQuantity(memoryRequest),
					},
					Max: corev1.ResourceList{
						corev1.ResourceCPU:    l.parseQuantity(cpuLimit),
						corev1.ResourceMemory: l.parseQuantity(memoryLimit),
					},
					Min: corev1.ResourceList{
						corev1.ResourceCPU:    l.parseQuantity(cpuRequest),
						corev1.ResourceMemory: l.parseQuantity(memoryRequest),
					},
				},
			},
		},
	}

	_, err := l.kubernetesClient.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create limit range: %w", err)
	}

	return nil
}

// GetLimitRange 获取命名空间的LimitRange
func (l *LimitRangeManager) GetLimitRange(namespace string) (*corev1.LimitRange, error) {
	ctx := context.Background()

	limitRanges, err := l.kubernetesClient.CoreV1().LimitRanges(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list limit ranges: %w", err)
	}

	if len(limitRanges.Items) == 0 {
		return nil, nil
	}

	return &limitRanges.Items[0], nil
}

// UpdateLimitRange 更新LimitRange
func (l *LimitRangeManager) UpdateLimitRange(namespace string, limitRange *corev1.LimitRange) error {
	ctx := context.Background()

	_, err := l.kubernetesClient.CoreV1().LimitRanges(namespace).Update(ctx, limitRange, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update limit range: %w", err)
	}

	return nil
}

// DeleteLimitRange 删除LimitRange
func (l *LimitRangeManager) DeleteLimitRange(namespace, name string) error {
	ctx := context.Background()

	err := l.kubernetesClient.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete limit range: %w", err)
	}

	return nil
}

// parseQuantity 解析资源数量字符串
func (l *LimitRangeManager) parseQuantity(quantityStr string) resource.Quantity {
	if quantityStr == "" {
		return resource.MustParse("0")
	}
	return resource.MustParse(quantityStr)
}

// LimitRangeInfo LimitRange信息
type LimitRangeInfo struct {
	Namespace       string            `json:"namespace"`
	DefaultRequest  map[string]string `json:"default_request"`
	DefaultLimit    map[string]string `json:"default_limit"`
	Min             map[string]string `json:"min"`
	Max             map[string]string `json:"max"`
}

// ConvertToInfo 转换为LimitRangeInfo
func (l *LimitRangeManager) ConvertToInfo(limitRange *corev1.LimitRange) *LimitRangeInfo {
	if limitRange == nil || len(limitRange.Spec.Limits) == 0 {
		return nil
	}

	limit := limitRange.Spec.Limits[0]

	info := &LimitRangeInfo{
		Namespace:       limitRange.Namespace,
		DefaultRequest:  make(map[string]string),
		DefaultLimit:    make(map[string]string),
		Min:             make(map[string]string),
		Max:             make(map[string]string),
	}

	// 转换DefaultRequest
	for k, v := range limit.DefaultRequest {
		info.DefaultRequest[string(k)] = v.String()
	}

	// 转换Default
	for k, v := range limit.Default {
		info.DefaultLimit[string(k)] = v.String()
	}

	// 转换Min
	for k, v := range limit.Min {
		info.Min[string(k)] = v.String()
	}

	// 转换Max
	for k, v := range limit.Max {
		info.Max[string(k)] = v.String()
	}

	return info
}
