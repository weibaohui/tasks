package config

// ConfigWorkspaceProvider implements domain.WorkspaceConfigProvider using config package.
type ConfigWorkspaceProvider struct{}

func (p *ConfigWorkspaceProvider) WorkspaceRoot() string {
	return GetAgentAIWorkSpaceRoot()
}
