package service

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecurityPolicyDetector 负责检测集群的安全策略配置
type SecurityPolicyDetector struct {
	clientset *kubernetes.Clientset
}

// NewSecurityPolicyDetector 创建新的安全策略检测器
func NewSecurityPolicyDetector(clientset *kubernetes.Clientset) *SecurityPolicyDetector {
	return &SecurityPolicyDetector{
		clientset: clientset,
	}
}

// RBACStatus RBAC状态信息
type RBACStatus struct {
	Enabled     bool   `json:"enabled"`
	APIVersions string `json:"api_versions"`
	RolesCount  int    `json:"roles_count"`
	Details     string `json:"details"`
}

// NetworkPolicyStatus 网络策略状态信息
type NetworkPolicyStatus struct {
	Enabled        bool   `json:"enabled"`
	PoliciesCount  int    `json:"policies_count"`
	CNIPlugin      string `json:"cni_plugin"`
	SupportsPolicy bool   `json:"supports_policy"`
	Details        string `json:"details"`
}

// PodSecurityStatus Pod安全状态信息
type PodSecurityStatus struct {
	Standard      string `json:"standard"`       // privileged, baseline, restricted, disabled
	AdmissionMode string `json:"admission_mode"` // PSA, PSP, none
	Details       string `json:"details"`
}

// AuditLoggingStatus 审计日志状态信息
type AuditLoggingStatus struct {
	Enabled    bool   `json:"enabled"`
	LogPath    string `json:"log_path"`
	PolicyFile string `json:"policy_file"`
	LogLevel   string `json:"log_level"` // none, metadata, request, request-response
	Details    string `json:"details"`
}

// DetectRBAC 检测RBAC是否启用
func (d *SecurityPolicyDetector) DetectRBAC(ctx context.Context) (*RBACStatus, error) {
	// 方法1: 检查API资源组是否存在
	serverGroups, err := d.clientset.Discovery().ServerGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get server groups: %w", err)
	}

	var rbacAPIVersion string
	for _, group := range serverGroups.Groups {
		if group.Name == "rbac.authorization.k8s.io" {
			rbacAPIVersion = group.PreferredVersion.GroupVersion
			break
		}
	}

	// 方法2: 检查API Server启动参数（通过Pod配置）
	// 使用更宽松的标签选择器，减少API调用
	apiServerPods, err := d.clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "component=kube-apiserver",
		Limit: 1, // 只需要获取一个pod即可
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get API server pods: %w", err)
	}

	var authMode string
	var details string
	if len(apiServerPods.Items) > 0 {
		pod := apiServerPods.Items[0]
		for _, container := range pod.Spec.Containers {
			for _, arg := range container.Command {
				if strings.Contains(arg, "--authorization-mode") {
					parts := strings.Split(arg, "=")
					if len(parts) > 1 {
						authMode = parts[1]
					}
				}
			}
		}
	}

	// 获取RBAC角色数量 - 使用限制来减少API调用
	roles, err := d.clientset.RbacV1().Roles("").List(ctx, metav1.ListOptions{
		Limit: 100, // 限制返回数量
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	clusterRoles, err := d.clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{
		Limit: 100, // 限制返回数量
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster roles: %w", err)
	}

	enabled := rbacAPIVersion != "" && strings.Contains(authMode, "RBAC")
	if rbacAPIVersion != "" && authMode == "" {
		// 如果无法获取启动参数，但API资源存在，则认为RBAC已启用
		enabled = true
	}

	if enabled {
		details = fmt.Sprintf("RBAC API版本: %s, 授权模式: %s", rbacAPIVersion, authMode)
		if authMode == "" {
			details = fmt.Sprintf("RBAC API版本: %s (无法获取授权模式参数)", rbacAPIVersion)
		}
	} else {
		details = "RBAC未启用"
		if rbacAPIVersion != "" {
			details += " (API资源存在但授权模式未包含RBAC)"
		} else {
			details += " (API资源不存在)"
		}
	}

	return &RBACStatus{
		Enabled:     enabled,
		APIVersions: rbacAPIVersion,
		RolesCount:  len(roles.Items) + len(clusterRoles.Items),
		Details:     details,
	}, nil
}

// DetectNetworkPolicy 检测网络策略是否启用
func (d *SecurityPolicyDetector) DetectNetworkPolicy(ctx context.Context) (*NetworkPolicyStatus, error) {
	// 方法1: 检查networking.k8s.io/v1 API资源
	serverGroups, err := d.clientset.Discovery().ServerGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get server groups: %w", err)
	}

	var networkingAPIVersion string
	for _, group := range serverGroups.Groups {
		if group.Name == "networking.k8s.io" {
			networkingAPIVersion = group.PreferredVersion.GroupVersion
			break
		}
	}

	// 方法2: 识别CNI插件
	cniPlugin, supportsPolicy, err := d.detectCNIPlugin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect CNI plugin: %w", err)
	}

	// 获取网络策略数量 - 使用限制来减少API调用
	var policiesCount int
	if networkingAPIVersion != "" {
		policies, err := d.clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{
			Limit: 100, // 限制返回数量
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list network policies: %w", err)
		}
		policiesCount = len(policies.Items)
	}

	enabled := networkingAPIVersion != "" && supportsPolicy
	details := fmt.Sprintf("网络API版本: %s, CNI插件: %s", networkingAPIVersion, cniPlugin)
	if !supportsPolicy {
		details += " (不支持网络策略)"
	}

	return &NetworkPolicyStatus{
		Enabled:        enabled,
		PoliciesCount:  policiesCount,
		CNIPlugin:      cniPlugin,
		SupportsPolicy: supportsPolicy,
		Details:        details,
	}, nil
}

// detectCNIPlugin 识别CNI插件
func (d *SecurityPolicyDetector) detectCNIPlugin(ctx context.Context) (string, bool, error) {
	// 检查kube-system命名空间中的CNI相关Pod - 使用限制来减少API调用
	pods, err := d.clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		Limit: 50, // 限制返回数量
	})
	if err != nil {
		return "", false, fmt.Errorf("failed to list pods: %w", err)
	}

	// 常见CNI插件标识
	cniPlugins := map[string]bool{
		"calico":  true,  // 支持网络策略
		"cilium":  true,  // 支持网络策略
		"weave":   true,  // 支持网络策略
		"flannel": false, // 不支持网络策略
		"canal":   true,  // 支持网络策略
		"antrea":  true,  // 支持网络策略
	}

	for _, pod := range pods.Items {
		for key, value := range pod.Labels {
			keyLower := strings.ToLower(key)
			valueLower := strings.ToLower(value)

			// 检查Pod标签中是否包含CNI插件标识
			for plugin := range cniPlugins {
				if strings.Contains(keyLower, plugin) || strings.Contains(valueLower, plugin) {
					return plugin, cniPlugins[plugin], nil
				}
			}
		}

		// 检查容器镜像名称
		for _, container := range pod.Spec.Containers {
			image := strings.ToLower(container.Image)
			for plugin := range cniPlugins {
				if strings.Contains(image, plugin) {
					return plugin, cniPlugins[plugin], nil
				}
			}
		}
	}

	return "未知", false, nil
}

// DetectPodSecurity 检测Pod安全策略
func (d *SecurityPolicyDetector) DetectPodSecurity(ctx context.Context) (*PodSecurityStatus, error) {
	// 检查Pod Security Admission (PSA) - 新版 - 使用限制来减少API调用
	namespaces, err := d.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		Limit: 100, // 限制返回数量
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	// 检查命名空间的PSS标签
	pssLevels := make(map[string]int)
	for _, ns := range namespaces.Items {
		if enforce, ok := ns.Labels["pod-security.kubernetes.io/enforce"]; ok {
			pssLevels[enforce]++
		}
		if audit, ok := ns.Labels["pod-security.kubernetes.io/audit"]; ok {
			pssLevels[audit]++
		}
		if warn, ok := ns.Labels["pod-security.kubernetes.io/warn"]; ok {
			pssLevels[warn]++
		}
	}

	var standard string
	var admissionMode string
	var details string

	// 检查PodSecurityPolicy (PSP) - 旧版
	serverGroups, err := d.clientset.Discovery().ServerGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get server groups: %w", err)
	}

	var pspAPIVersion string
	for _, group := range serverGroups.Groups {
		for _, version := range group.Versions {
			if version.GroupVersion == "policy/v1beta1" {
				pspAPIVersion = version.GroupVersion
				break
			}
		}
		if pspAPIVersion != "" {
			break
		}
	}

	// 判断使用哪种模式
	if len(pssLevels) > 0 {
		// PSA模式
		admissionMode = "PSA"
		// 优先级: restricted > baseline > privileged
		if pssLevels["restricted"] > 0 {
			standard = "restricted"
		} else if pssLevels["baseline"] > 0 {
			standard = "baseline"
		} else if pssLevels["privileged"] > 0 {
			standard = "privileged"
		} else {
			standard = "baseline" // 默认值
		}

		details = fmt.Sprintf("使用Pod Security Admission, enforce级别: %s", standard)
		if pssLevels["audit"] > 0 {
			details += fmt.Sprintf(", audit级别: %d个命名空间", pssLevels["audit"])
		}
		if pssLevels["warn"] > 0 {
			details += fmt.Sprintf(", warn级别: %d个命名空间", pssLevels["warn"])
		}
	} else if pspAPIVersion != "" {
		// PSP模式
		admissionMode = "PSP"
		standard = "baseline" // PSP没有明确的级别概念，使用baseline作为默认值
		details = fmt.Sprintf("使用Pod Security Policy (已废弃), API版本: %s", pspAPIVersion)
	} else {
		// 无Pod安全策略
		admissionMode = "none"
		standard = "disabled"
		details = "未配置Pod安全策略"
	}

	return &PodSecurityStatus{
		Standard:      standard,
		AdmissionMode: admissionMode,
		Details:       details,
	}, nil
}

// DetectAuditLogging 检测审计日志是否启用
func (d *SecurityPolicyDetector) DetectAuditLogging(ctx context.Context) (*AuditLoggingStatus, error) {
	// 检查API Server启动参数 - 使用限制来减少API调用
	apiServerPods, err := d.clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "component=kube-apiserver",
		Limit: 1, // 只需要获取一个pod即可
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get API server pods: %w", err)
	}

	var enabled bool
	var logPath string
	var policyFile string
	var logLevel string

	if len(apiServerPods.Items) > 0 {
		pod := apiServerPods.Items[0]
		for _, container := range pod.Spec.Containers {
			for _, arg := range container.Command {
				if strings.Contains(arg, "--audit-log-path") {
					enabled = true
					parts := strings.Split(arg, "=")
					if len(parts) > 1 {
						logPath = parts[1]
					}
				}
				if strings.Contains(arg, "--audit-policy-file") {
					parts := strings.Split(arg, "=")
					if len(parts) > 1 {
						policyFile = parts[1]
					}
				}
				if strings.Contains(arg, "--audit-log-maxage") ||
					strings.Contains(arg, "--audit-log-maxbackup") ||
					strings.Contains(arg, "--audit-log-maxsize") {
					// 这些参数表明审计日志已详细配置
					logLevel = "request-response"
				}
			}
		}
	}

	// 如果启用了审计日志但没有明确设置日志级别，默认为metadata
	if enabled && logLevel == "" {
		logLevel = "metadata"
	}

	var details string
	if enabled {
		details = fmt.Sprintf("审计日志已启用, 路径: %s", logPath)
		if policyFile != "" {
			details += fmt.Sprintf(", 策略文件: %s", policyFile)
		}
		details += fmt.Sprintf(", 日志级别: %s", logLevel)
	} else {
		details = "审计日志未启用"
	}

	return &AuditLoggingStatus{
		Enabled:    enabled,
		LogPath:    logPath,
		PolicyFile: policyFile,
		LogLevel:   logLevel,
		Details:    details,
	}, nil
}
