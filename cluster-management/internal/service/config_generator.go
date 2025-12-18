package service

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
)

// MachineInfo 机器信息结构体
type MachineInfo struct {
	ID               uuid.UUID
	Name             string
	IPAddress        string
	InternalAddress  string
	User             string
	Role             string
	RegistryAddress  string
	ImageRepo        string
}

// KubernetesConfig Kubernetes 配置
type KubernetesConfig struct {
	Version         string
	ImageRepo       string
	ContainerManager string
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	Plugin         string
	PodsCIDR       string
	ServiceCIDR    string
}

// CreateClusterRequest 创建集群请求
type CreateClusterRequest struct {
	ClusterName        string
	MachineIDs         []uuid.UUID
	Kubernetes         KubernetesConfig
	Network            NetworkConfig
	ArtifactPath       string
	WithPackages       bool
	AutoApprove        bool
}

// ConfigGenerator 配置生成器
type ConfigGenerator struct{}

// NewConfigGenerator 创建配置生成器
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

// GenerateConfig 生成 KubeKey 配置文件
func (g *ConfigGenerator) GenerateConfig(req CreateClusterRequest, machines []*model.Machine) (string, error) {
	// 转换机器信息
	machineInfos := g.convertMachines(machines)

	// 组织节点数据
	hosts := make([]map[string]interface{}, 0, len(machineInfos))
	masters := make([]string, 0)
	workers := make([]string, 0)
	etcds := make([]string, 0)

	for _, machine := range machineInfos {
		host := map[string]interface{}{
			"name":             machine.Name,
			"address":          machine.IPAddress,
			"internalAddress":  machine.InternalAddress,
			"user":             machine.User,
			"role":             machine.Role,
		}

		// 添加密码（这里应该解密后使用，实际使用中需要从加密服务获取）
		// host["password"] = decryptedPassword

		hosts = append(hosts, host)

		// 按角色分组
		switch machine.Role {
		case "master":
			masters = append(masters, machine.Name)
		case "worker":
			workers = append(workers, machine.Name)
		case "etcd":
			etcds = append(etcds, machine.Name)
		}
	}

	// 获取注册表机器
	var registryHost string
	var registryPort string
	var imageRepo string

	for _, machine := range machineInfos {
		if machine.Role == "registry" && machine.RegistryAddress != "" {
			registryHost = machine.IPAddress
			registryPort = "5000" // 默认端口
			imageRepo = machine.ImageRepo
			break
		}
	}

	// 如果没有专门的 registry 机器，使用第一个 master
	if registryHost == "" && len(masters) > 0 {
		for _, machine := range machineInfos {
			if machine.Name == masters[0] {
				registryHost = machine.IPAddress
				registryPort = "5000"
				imageRepo = machine.ImageRepo
				break
			}
		}
	}

	// 默认值
	if imageRepo == "" {
		imageRepo = "kubesphere"
	}

	if req.Network.PodsCIDR == "" {
		req.Network.PodsCIDR = "10.233.64.0/18"
	}
	if req.Network.ServiceCIDR == "" {
		req.Network.ServiceCIDR = "10.233.0.0/18"
	}
	if req.Network.Plugin == "" {
		req.Network.Plugin = "calico"
	}
	if req.Kubernetes.Version == "" {
		req.Kubernetes.Version = "v1.28.0"
	}
	if req.Kubernetes.ContainerManager == "" {
		req.Kubernetes.ContainerManager = "containerd"
	}

	// 构建配置数据
	configData := map[string]interface{}{
		"clusterName":     req.ClusterName,
		"hosts":           hosts,
		"masters":         masters,
		"workers":         workers,
		"etcds":           etcds,
		"kubernetesVersion": req.Kubernetes.Version,
		"imageRepo":       imageRepo,
		"containerManager": req.Kubernetes.ContainerManager,
		"networkPlugin":   req.Network.Plugin,
		"podsCIDR":        req.Network.PodsCIDR,
		"serviceCIDR":     req.Network.ServiceCIDR,
		"registryHost":    registryHost,
		"registryPort":    registryPort,
		"hasRegistry":     registryHost != "",
	}

	// 渲染模板
	return g.renderTemplate(configData)
}

// convertMachines 转换机器模型
func (g *ConfigGenerator) convertMachines(machines []*model.Machine) []MachineInfo {
	infos := make([]MachineInfo, 0, len(machines))
	for _, machine := range machines {
		infos = append(infos, MachineInfo{
			ID:               machine.ID,
			Name:             machine.Name,
			IPAddress:        machine.IPAddress,
			InternalAddress:  machine.InternalAddress,
			User:             machine.User,
			Role:             machine.Role,
			RegistryAddress:  machine.RegistryAddress,
			ImageRepo:        machine.ImageRepo,
		})
	}
	return infos
}

// renderTemplate 渲染配置模板
func (g *ConfigGenerator) renderTemplate(data map[string]interface{}) (string, error) {
	tmpl := `apiVersion: kubekey.kubesphere.io/v1alpha2
kind: Cluster
metadata:
  name: {{.clusterName}}
spec:
  hosts:
{{- range .hosts }}
  - name: {{.name}}
    address: {{.address}}
    internalAddress: {{.internalAddress}}
    user: {{.user}}
    role: [{{.role}}]
{{- end }}
  roleGroups:
{{- if .etcds }}
    etcd:
{{- range .etcds }}
      - {{.}}
{{- end }}
{{- end }}
{{- if .masters }}
    master:
{{- range .masters }}
      - {{.}}
{{- end }}
{{- end }}
{{- if .workers }}
    worker:
{{- range .workers }}
      - {{.}}
{{- end }}
{{- end }}
  controlPlaneEndpoint:
    domain: lb.kubesphere.local
    address: ""
    port: 6443
  kubernetes:
    version: {{.kubernetesVersion}}
    imageRepo: "{{.imageRepo}}"
    etcd:
      type: kubekey
    containerManager: {{.containerManager}}
  network:
    plugin: {{.networkPlugin}}
    kubePodsCIDR: {{.podsCIDR}}
    kubeServiceCIDR: {{.serviceCIDR}}
{{- if .hasRegistry }}
  registry:
    type: kubekey
    registries:
      - {{.registryHost}}:{{.registryPort}}
{{- else }}
  registry:
    type: kubekey
    insecureRegistry: false
    registryMirrors: []
    privateRegistry: ""
{{- end }}
  addons: []
`

	t, err := template.New("config").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
