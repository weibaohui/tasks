package workspace

import "os"

// OSWorkspaceManager implements domain.WorkspaceManager using os package.
type OSWorkspaceManager struct{}

func (m *OSWorkspaceManager) CreateWorkspace(path string) error {
	return os.MkdirAll(path, 0755)
}

func (m *OSWorkspaceManager) RemoveWorkspace(path string) error {
	return os.RemoveAll(path)
}
