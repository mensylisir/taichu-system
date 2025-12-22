package service

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesService 定义Kubernetes客户端服务的接口
type KubernetesService interface {
	GetClientset(kubeconfig string) (*kubernetes.Clientset, error)
}

// kubernetesServiceImpl 实现KubernetesService接口
type kubernetesServiceImpl struct {
	timeout time.Duration
}

// NewKubernetesService 创建新的KubernetesService实例
func NewKubernetesService() KubernetesService {
	return &kubernetesServiceImpl{
		timeout: 30 * time.Second, // 默认超时时间
	}
}

// GetClientset 根据kubeconfig创建Kubernetes客户端
func (k *kubernetesServiceImpl) GetClientset(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}

	config.Timeout = k.timeout

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
