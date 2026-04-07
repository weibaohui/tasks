package domain

import "context"

// ReplicaCleanupService 分身清理服务接口
// 定义在 domain 层，由 infrastructure 层实现
// 这样 domain 层不依赖 infrastructure，但可以声明它需要的服务
type ReplicaCleanupService interface {
	// CleanupReplica 清理分身和工作区
	CleanupReplica(ctx context.Context, replicaAgentCode, workspacePath string) error
}