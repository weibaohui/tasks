package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/weibh/taskmanager/domain"
)

var (
	ErrConversationRecordNotFound = errors.New("conversation record not found")
)

type CreateConversationRecordCommand struct {
	TraceID          string
	SpanID           string
	ParentSpanID     string
	EventType        string
	Timestamp        *time.Time
	SessionKey       string
	Role             string
	Content          string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	ReasoningTokens  int
	CachedTokens     int
	UserCode         string
	AgentCode        string
	ChannelCode      string
	ChannelType      string
}

type ListConversationRecordsQuery struct {
	TraceID     string
	SessionKey  string
	UserCode    string
	AgentCode   string
	ChannelCode string
	EventType   string
	Role        string
	Limit       int
	Offset      int
}

type GetConversationStatsQuery struct {
	StartTime    *time.Time
	EndTime      *time.Time
	AgentCodes   []string
	ChannelCodes []string
	Roles        []string
}

type ConversationRecordApplicationService struct {
	recordRepo   domain.ConversationRecordRepository
	idGenerator  domain.IDGenerator
	defaultLimit int
	maxLimit     int
}

func NewConversationRecordApplicationService(
	recordRepo domain.ConversationRecordRepository,
	idGenerator domain.IDGenerator,
) *ConversationRecordApplicationService {
	return &ConversationRecordApplicationService{
		recordRepo:   recordRepo,
		idGenerator:  idGenerator,
		defaultLimit: 100,
		maxLimit:     500,
	}
}

func (s *ConversationRecordApplicationService) CreateRecord(ctx context.Context, cmd CreateConversationRecordCommand) (*domain.ConversationRecord, error) {
	record, err := domain.NewConversationRecord(
		domain.NewConversationRecordID(s.idGenerator.Generate()),
		cmd.TraceID,
		cmd.EventType,
	)
	if err != nil {
		return nil, err
	}
	record.SetSpan(cmd.SpanID, cmd.ParentSpanID)
	record.SetScope(cmd.SessionKey, cmd.UserCode, cmd.AgentCode, cmd.ChannelCode, cmd.ChannelType)
	record.SetMessage(cmd.Role, cmd.Content)
	record.SetTokenUsage(cmd.PromptTokens, cmd.CompletionTokens, cmd.TotalTokens, cmd.ReasoningTokens, cmd.CachedTokens)
	if cmd.Timestamp != nil {
		record.SetTimestamp(*cmd.Timestamp)
	}
	if err := s.recordRepo.Save(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to save conversation record: %w", err)
	}
	return record, nil
}

func (s *ConversationRecordApplicationService) GetRecord(ctx context.Context, id string) (*domain.ConversationRecord, error) {
	record, err := s.recordRepo.FindByID(ctx, domain.NewConversationRecordID(id))
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrConversationRecordNotFound
	}
	return record, nil
}

func (s *ConversationRecordApplicationService) ListRecords(ctx context.Context, query ListConversationRecordsQuery) ([]*domain.ConversationRecord, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = s.defaultLimit
	}
	if limit > s.maxLimit {
		limit = s.maxLimit
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	filter := domain.ConversationRecordListFilter{
		TraceID:     query.TraceID,
		SessionKey:  query.SessionKey,
		UserCode:    query.UserCode,
		AgentCode:   query.AgentCode,
		ChannelCode: query.ChannelCode,
		EventType:   query.EventType,
		Role:        query.Role,
		Limit:       limit,
		Offset:      offset,
	}
	return s.recordRepo.List(ctx, filter)
}

func (s *ConversationRecordApplicationService) GetStats(ctx context.Context, query GetConversationStatsQuery) (*domain.ConversationStats, error) {
	filter := domain.ConversationStatsFilter{
		StartTime:    query.StartTime,
		EndTime:      query.EndTime,
		AgentCodes:   query.AgentCodes,
		ChannelCodes: query.ChannelCodes,
		Roles:        query.Roles,
	}
	return s.recordRepo.GetStats(ctx, filter)
}
