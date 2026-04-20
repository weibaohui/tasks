package platform

import (
	"fmt"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/platform/atomgit"
	"github.com/weibh/taskmanager/infrastructure/platform/github"
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

// NewProviderByRepo 根据仓库 URL 自动检测平台类型并创建对应的 Provider
func NewProviderByRepo(repo string) (domain.PlatformProvider, error) {
	platformType := domain.DetectPlatformType(repo)
	return NewProvider(platformType)
}

// MustNewProviderByRepo 根据仓库 URL 自动检测平台类型并创建对应的 Provider
// 如果不支持则 panic
func MustNewProviderByRepo(repo string) domain.PlatformProvider {
	provider, err := NewProviderByRepo(repo)
	if err != nil {
		panic(err)
	}
	return provider
}
