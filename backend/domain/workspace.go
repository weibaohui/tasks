package domain

// WorkspaceConfigProvider provides workspace root path configuration.
type WorkspaceConfigProvider interface {
	WorkspaceRoot() string
}

// WorkspaceManager manages workspace directory lifecycle.
type WorkspaceManager interface {
	CreateWorkspace(path string) error
	RemoveWorkspace(path string) error
}
