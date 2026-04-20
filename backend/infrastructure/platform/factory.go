package platform

import (
	"fmt"

	"taskmanager/backend/domain"
	"taskmanager/backend/infrastructure/platform/atomgit"
	"taskmanager/backend/infrastructure/platform/github"
)

// NewProvider 根据平台类型创建对应的 Provider
func NewProvider(platformType domain.PlatformType) (domain.PlatformProvider, error) {
	switch platformType {
	case domain.PlatformTypeGitHub:
		return github.NewProvider(), nil
	case domain.PlatformTypeAtomGit:
		return atomgit.NewProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported platform type: %s", platformType)
	}
}

// MustNewProvider 根据平台类型创建对应的 Provider，如果不支持则 panic
func MustNewProvider(platformType domain.PlatformType) domain.PlatformProvider {
	provider, err := NewProvider(platformType)
	if err != nil {
		panic(err)
	}
	return provider
}
