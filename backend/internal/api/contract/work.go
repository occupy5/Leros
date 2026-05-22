package contract

import "context"

// WorkService 定义工作台服务接口
type WorkService interface {
	// NewMessage 首页新建消息接口，原子创建 Project + Task + Session 并分配 AgentWorker
	NewMessage(ctx context.Context, req *NewMessageRequest) (*NewMessageResponse, error)
}
