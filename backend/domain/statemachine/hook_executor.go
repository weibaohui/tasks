package statemachine

import "context"

// HookContext holds context information for hook execution.
type HookContext struct {
	RequirementID  string
	StateMachineID string
	FromState      string
	ToState        string
	Trigger        string
	TriggerID      string
	HookName       string
	HookType       string
	Metadata       map[string]interface{}
}

// HookExecutor executes transition hooks asynchronously.
type HookExecutor interface {
	ExecuteHooks(ctx context.Context, hooks []TransitionHook, hookCtx HookContext)
}

// metadataContextKey is the context key for storing metadata.
type metadataContextKey struct{}

// WithMetadata stores metadata in context for hook template variable substitution.
func WithMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	return context.WithValue(ctx, metadataContextKey{}, metadata)
}

// MetadataFromContext retrieves metadata from context.
func MetadataFromContext(ctx context.Context) map[string]interface{} {
	if metadata, ok := ctx.Value(metadataContextKey{}).(map[string]interface{}); ok {
		return metadata
	}
	return nil
}
