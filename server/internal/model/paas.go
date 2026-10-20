package model

import "time"

// PaaS 服务器状态。
const (
	PaasServerOffline = 0 // 离线
	PaasServerOnline  = 1 // 在线
	PaasServerIniting = 2 // 环境初始化中
)

// 应用状态。
const (
	PaasAppUndeployed = 0 // 未部署
	PaasAppRunning    = 1 // 运行中
	PaasAppStopped    = 2 // 已停止
	PaasAppDeploying  = 3 // 部署中
)

// SSH 认证方式。
const (
	PaasAuthPassword   = 1
	PaasAuthPrivateKey = 2
)

// PaasServer PaaS 纳管的用户服务器。
type PaasServer struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	Name        string    `gorm:"size:64;index"`
	IP          string    `gorm:"size:64;uniqueIndex"`
	SSHPort     int       `gorm:"default:22"`
	SSHUser     string    `gorm:"size:32;default:root"`
	AuthType    int       `gorm:"default:1"`       // 1-密码 2-私钥
	AuthContent string    `gorm:"type:text"`       // AES-GCM 加密后的凭证
	Status      int       `gorm:"default:0;index"` // 0-离线 1-在线 2-初始化中
	DockerVer   string    `gorm:"size:64"`         // 检测后回填
	LastError   string    `gorm:"type:text"`       // 最近一次初始化/操作失败原因
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
}

// TableName 表名。
func (PaasServer) TableName() string { return "paas_servers" }

// PaasApp 部署在纳管服务器上的容器应用。
type PaasApp struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	ServerID      uint64 `gorm:"index"`
	Name          string `gorm:"size:64;uniqueIndex"` // 同时作为容器名，全局唯一
	Image         string `gorm:"size:255"`
	ContainerPort int
	HostPort      int
	Env           string    `gorm:"type:text"` // JSON 数组 ["K=V", ...]
	Volumes       string    `gorm:"type:text"` // JSON 数组 ["/host:/ctr[:ro]", ...]
	Restart       string    `gorm:"size:32;default:always"`
	ContainerID   string    `gorm:"size:64"`
	Status        int       `gorm:"default:0;index"` // 0-未部署 1-运行中 2-已停止 3-部署中
	LastError     string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time
}

// TableName 表名。
func (PaasApp) TableName() string { return "paas_apps" }

// PaasDomain 应用绑定的访问域名，由 Caddy 自动签发 HTTPS 证书。
type PaasDomain struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	AppID     uint64    `gorm:"index"`
	Domain    string    `gorm:"size:253;uniqueIndex"`
	CreatedAt time.Time `gorm:"index"`
}

// TableName 表名。
func (PaasDomain) TableName() string { return "paas_domains" }
