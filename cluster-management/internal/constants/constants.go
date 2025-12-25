package constants

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusCompleted = "completed"
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusDeleted   = "deleted"
)

const (
	EventTypeCreate = "create"
	EventTypeUpdate = "update"
	EventTypeDelete = "delete"
	EventTypeError  = "error"
)

const (
	ResourceTypeTenant       = "tenant"
	ResourceTypeCluster      = "cluster"
	ResourceTypeEnvironment  = "environment"
	ResourceTypeApplication = "application"
	ResourceTypeBackup       = "backup"
	ResourceTypeSchedule     = "schedule"
	ResourceTypeRestore      = "restore"
	ResourceTypeMachine      = "machine"
	ResourceTypeNode         = "node"
	ResourceTypeUser         = "user"
	ResourceTypeRole         = "role"
)

const (
	BackupTypeFull    = "full"
	BackupTypeIncremental = "incremental"
)

const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusSuccess   = "success"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

const (
	ImportStatusPending   = "pending"
	ImportStatusRunning   = "running"
	ImportStatusSuccess   = "success"
	ImportStatusFailed    = "failed"
)

const (
	ClusterStatusCreating  = "creating"
	ClusterStatusRunning   = "running"
	ClusterStatusStopped   = "stopped"
	ClusterStatusError     = "error"
	ClusterStatusDeleting  = "deleting"
)

const (
	ApplicationStatusCreating = "creating"
	ApplicationStatusRunning  = "running"
	ApplicationStatusStopped  = "stopped"
	ApplicationStatusError    = "error"
	ApplicationStatusDeleting = "deleting"
)

const (
	BackupStatusPending   = "pending"
	BackupStatusRunning   = "running"
	BackupStatusSuccess   = "success"
	BackupStatusFailed    = "failed"
)

const (
	RestoreStatusPending   = "pending"
	RestoreStatusRunning   = "running"
	RestoreStatusSuccess   = "success"
	RestoreStatusFailed    = "failed"
)

const (
	ScheduleStatusActive   = "active"
	ScheduleStatusInactive = "inactive"
	ScheduleStatusPaused   = "paused"
)

const (
	ExpansionStatusPending    = "pending"
	ExpansionStatusRunning    = "running"
	ExpansionStatusInProgress = "in_progress"
	ExpansionStatusSuccess    = "success"
	ExpansionStatusFailed     = "failed"
)

const (
	AuditResultSuccess = "success"
	AuditResultFailed  = "failed"
	AuditResultError   = "error"
)

const (
	AuthHeaderRequired     = "Authorization header is required"
	AuthHeaderInvalidFormat = "Authorization header format must be Bearer {token}"
	AuthTokenInvalidOrExpired = "Invalid or expired token"
	AuthTokenInvalid        = "Invalid token"
	AuthUserRoleNotFound    = "User role not found"
	AuthInvalidUserRoleFormat = "Invalid user role format"
	AuthInsufficientPermissions = "Insufficient permissions"
)
