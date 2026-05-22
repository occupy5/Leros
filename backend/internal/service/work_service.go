package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/insmtx/Leros/backend/internal/api/auth"
	"github.com/insmtx/Leros/backend/internal/api/contract"
	"github.com/insmtx/Leros/backend/internal/infra/db"
	eventbus "github.com/insmtx/Leros/backend/internal/infra/mq"
	"github.com/insmtx/Leros/backend/internal/worker/protocol"
	"github.com/insmtx/Leros/backend/pkg/dm"
	"github.com/insmtx/Leros/backend/types"
	"github.com/ygpkg/yg-go/encryptor/snowflake"
	"github.com/ygpkg/yg-go/logs"
)

var _ contract.WorkService = (*workService)(nil)

type workService struct {
	db       *gorm.DB
	eventbus eventbus.EventBus
	inferrer AssistantInferrer
}

func NewWorkService(database *gorm.DB, eventbus eventbus.EventBus, inferrer AssistantInferrer) contract.WorkService {
	return &workService{
		db:       database,
		eventbus: eventbus,
		inferrer: inferrer,
	}
}

func (s *workService) NewMessage(ctx context.Context, req *contract.NewMessageRequest) (*contract.NewMessageResponse, error) {
	if req.Content == "" {
		return nil, errors.New("content is required")
	}

	caller, _ := auth.FromContext(ctx)
	if caller == nil || caller.Uin == 0 || caller.OrgID == 0 {
		return nil, errors.New("user not authenticated or org not set")
	}

	var project *types.Project

	if req.ProjectID != "" {
		p, err := db.GetProjectByPublicID(ctx, s.db, caller.OrgID, req.ProjectID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, errors.New("project not found")
		}
		project = p
	} else {
		runes := []rune(req.Content)
		title := string(runes)
		if len(runes) > 50 {
			title = string(runes[:50])
		}

		projectID := fmt.Sprintf("prj_%s", snowflake.GenerateIDBase58())
		project = &types.Project{
			PublicID:    projectID,
			OrgID:       caller.OrgID,
			OwnerID:     caller.Uin,
			Name:        title,
			Description: "",
			Status:      string(types.ProjectStatusActive),
		}
		if err := db.CreateProject(ctx, s.db, project); err != nil {
			return nil, fmt.Errorf("create project: %w", err)
		}

		if err := db.CreateProjectMember(ctx, s.db, &types.ProjectMember{
			ProjectID:  project.ID,
			MemberID:   caller.Uin,
			MemberType: types.MemberTypeUser,
			MemberRole: types.MemberRoleOwner,
		}); err != nil {
			logs.WarnContextf(ctx, "create project member failed: %v", err)
		}
	}

	projectSession, err := db.GetProjectSession(ctx, s.db, project.ID)
	if err != nil {
		return nil, fmt.Errorf("get project session: %w", err)
	}
	if projectSession == nil {
		projectSessionID := fmt.Sprintf("sess_%s", snowflake.GenerateIDBase58())
		projectSession = &types.Session{
			PublicID:             projectSessionID,
			Type:                 types.SessionTypeProject,
			Uin:                  caller.Uin,
			OrgID:                caller.OrgID,
			AssistantID:          req.AssistantID,
			AllocatedAssistantID: req.AssistantID,
			ProjectID:            &project.ID,
			Status:               string(types.SessionStatusActive),
			Title:                "项目协作",
		}
		if err := db.CreateSession(ctx, s.db, projectSession); err != nil {
			return nil, fmt.Errorf("create project session: %w", err)
		}
	}

	var task *types.Task

	if req.TaskID != "" {
		t, err := db.GetTaskByPublicID(ctx, s.db, req.TaskID)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return nil, errors.New("task not found")
		}
		task = t
	} else {
		runes := []rune(req.Content)
		taskTitle := string(runes)
		if len(runes) > 50 {
			taskTitle = string(runes[:50])
		}

		taskID := fmt.Sprintf("task_%s", snowflake.GenerateIDBase58())
		task = &types.Task{
			PublicID:    taskID,
			OrgID:       caller.OrgID,
			OwnerID:     caller.Uin,
			ProjectID:   project.ID,
			TaskType:    types.TaskTypeGeneral,
			Title:       taskTitle,
			Description: req.Content,
			Status:      string(types.TaskStatusCreated),
		}
		if err := db.CreateTask(ctx, s.db, task); err != nil {
			return nil, fmt.Errorf("create task: %w", err)
		}
	}

	taskSessionID := fmt.Sprintf("sess_%s", snowflake.GenerateIDBase58())
	taskSession := &types.Session{
		PublicID:             taskSessionID,
		Type:                 types.SessionTypeTask,
		Uin:                  caller.Uin,
		OrgID:                caller.OrgID,
		AssistantID:          req.AssistantID,
		AllocatedAssistantID: req.AssistantID,
		ProjectID:            &project.ID,
		TaskID:               &task.ID,
		Status:               string(types.SessionStatusActive),
		Title:                task.Title,
	}
	if err := db.CreateSession(ctx, s.db, taskSession); err != nil {
		return nil, fmt.Errorf("create task session: %w", err)
	}

	task.SessionID = &taskSession.ID
	if err := s.db.WithContext(ctx).Model(task).Update("session_id", taskSession.ID).Error; err != nil {
		logs.WarnContextf(ctx, "update task session_id failed: %v", err)
	}

	sequence, err := db.GetNextSequence(ctx, s.db, taskSession.ID)
	if err != nil {
		return nil, err
	}

	msgType := req.MessageType
	if msgType == "" {
		msgType = string(types.MessageTypeText)
	}

	message := &types.SessionMessage{
		SessionID:   taskSession.ID,
		Role:        string(types.MessageRoleUser),
		Content:     req.Content,
		MessageType: msgType,
		Status:      string(types.MessageStatusPending),
		Sequence:    sequence,
		Timestamp:   time.Now().UnixMilli(),
	}
	if err := db.CreateMessage(ctx, s.db, message); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	now := time.Now()
	if err := db.IncrementMessageCount(ctx, s.db, taskSession.ID); err != nil {
		return nil, err
	}
	if err := db.UpdateLastMessageAt(ctx, s.db, taskSession.ID, now); err != nil {
		return nil, err
	}

	if taskSession.OrgID > 0 {
		topic, err := dm.SessionMessageRequestSubject(taskSession.OrgID, taskSession.PublicID)
		if err != nil {
			logs.WarnContextf(ctx, "failed to build message request subject: %v", err)
		} else {
			if err := s.eventbus.Publish(ctx, topic, message); err != nil {
				logs.WarnContextf(ctx, "failed to publish message to eventbus: %v", err)
			}
		}
	}

	if err := s.publishWorkerTask(ctx, taskSession, message); err != nil {
		return nil, err
	}

	return &contract.NewMessageResponse{
		ProjectID:   project.PublicID,
		TaskID:      task.PublicID,
		SessionID:   taskSession.PublicID,
		MessageID:   fmt.Sprintf("%d", message.ID),
		AssistantID: taskSession.AllocatedAssistantID,
	}, nil
}

func (s *workService) publishWorkerTask(ctx context.Context, session *types.Session, message *types.SessionMessage) error {
	caller, _ := auth.FromContext(ctx)
	orgID := session.OrgID
	if orgID == 0 && caller != nil {
		orgID = caller.OrgID
	}

	if session.AssistantID == 0 && session.AllocatedAssistantID == 0 && s.inferrer != nil {
		assignedAssistantID := s.inferrer.InferAssignedAssistantID(ctx, orgID, string(session.Type))
		if assignedAssistantID > 0 {
			session.AllocatedAssistantID = assignedAssistantID
			if err := db.UpdateAllocatedAssistantID(ctx, s.db, session.ID, assignedAssistantID); err != nil {
				return fmt.Errorf("failed to update allocated_assistant_id: %w", err)
			}
		}
	}

	if session.AllocatedAssistantID == 0 {
		logs.DebugContextf(ctx, "Skipping task publish: no worker allocated for session %s", session.PublicID)
		return nil
	}

	topic, err := dm.WorkerTaskSubject(orgID, session.AllocatedAssistantID)
	if err != nil {
		return fmt.Errorf("failed to construct worker task topic: %w", err)
	}

	messagePayload := protocol.WorkerTaskMessage{
		ID:        fmt.Sprintf("msg_%d_%d", session.ID, message.Sequence),
		Type:      protocol.MessageTypeWorkerTask,
		CreatedAt: time.Now().UTC(),
		Trace: protocol.TraceContext{
			TraceID:   session.PublicID,
			RequestID: fmt.Sprintf("req_%d", message.ID),
			TaskID:    fmt.Sprintf("task_%d", message.ID),
		},
		Route: protocol.RouteContext{
			OrgID:     orgID,
			SessionID: session.PublicID,
			WorkerID:  session.AllocatedAssistantID,
		},
		Body: protocol.WorkerTaskBody{
			TaskType: protocol.TaskTypeAgentRun,
			Actor: protocol.ActorContext{
				UserID:      fmt.Sprintf("%d", session.Uin),
				DisplayName: "",
				Channel:     "session",
			},
			Input: protocol.TaskInput{
				Type: protocol.InputTypeMessage,
			},
		},
		Metadata: map[string]any{
			"session_id":   session.PublicID,
			"message_type": message.MessageType,
			"sequence":     message.Sequence,
			"timestamp":    message.Timestamp,
		},
	}

	if err := s.eventbus.Publish(ctx, topic, messagePayload); err != nil {
		logs.ErrorContextf(ctx, "Failed to publish message to assistant %d: %v", session.AllocatedAssistantID, err)
		return fmt.Errorf("failed to publish message to assistant: %w", err)
	}
	logs.DebugContextf(ctx, "Published message to topic %s: session_id=%s sequence=%d", topic, session.PublicID, message.Sequence)
	return nil
}
