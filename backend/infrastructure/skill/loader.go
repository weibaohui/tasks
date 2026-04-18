/**
 * Skills Loader
 * 技能加载器 - 从多个目录加载技能，支持灵活的搜索路径
 */
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/config"
	"gopkg.in/yaml.v3"
)

// SkillLoaderFunc 技能加载函数类型
type SkillLoaderFunc func(name string) string

// ensure *SkillsLoader implements domain.SkillsLoader
var _ domain.SkillsLoader = (*SkillsLoader)(nil)

// SkillsLoader 技能加载器
type SkillsLoader struct {
	searchPaths []string // 搜索路径列表，按优先级排序（后面的覆盖前面的）
}

// NewSkillsLoader 创建技能加载器，使用默认搜索路径
func NewSkillsLoader(workspace string) *SkillsLoader {
	paths := detectDefaultSearchPaths(workspace)
	return &SkillsLoader{
		searchPaths: paths,
	}
}

// NewSkillsLoaderWithPaths 创建技能加载器，使用指定的搜索路径
// searchPaths 列表中，靠后的路径优先级更高（同名技能会被后面的覆盖）
func NewSkillsLoaderWithPaths(searchPaths []string) *SkillsLoader {
	// 过滤掉不存在的路径
	validPaths := make([]string, 0, len(searchPaths))
	for _, p := range searchPaths {
		if p != "" && isDir(p) {
			validPaths = append(validPaths, p)
		}
	}
	return &SkillsLoader{
		searchPaths: validPaths,
	}
}

// GetSearchPaths 返回当前配置的搜索路径
func (s *SkillsLoader) GetSearchPaths() []string {
	return s.searchPaths
}

// detectDefaultSearchPaths 检测默认搜索路径
// 优先级从低到高，后面的会覆盖前面的同名技能
func detectDefaultSearchPaths(workspace string) []string {
	var paths []string

	// 1. 内置技能目录（从可执行文件所在目录推断）
	if exePath := detectExecutableDir(); exePath != "" {
		builtinPath := filepath.Join(exePath, "skills")
		if isDir(builtinPath) {
			paths = append(paths, builtinPath)
		}
	}

	// 3. 用户目录下的 Skills
	if homeDir := config.GetHomeDir(); homeDir != "" {
		userPath := filepath.Join(homeDir, ".taskmanager", "skills")
		if isDir(userPath) {
			paths = append(paths, userPath)
		}
	}

	// 4. 工作区技能目录
	if workspace != "" {
		workspacePath := filepath.Join(workspace, "skills")
		if isDir(workspacePath) {
			paths = append(paths, workspacePath)
		}
	}

	// 5. 当前工作目录下的 Skills（最低优先级）
	cwd, err := os.Getwd()
	if err == nil {
		cwdPath := filepath.Join(cwd, "skills")
		if isDir(cwdPath) {
			paths = append(paths, cwdPath)
		}
	}

	return paths
}

// detectExecutableDir 检测可执行文件所在目录
func detectExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// ListSkills 列出所有可用技能（从所有搜索路径合并）
func (s *SkillsLoader) ListSkills() []domain.SkillInfo {
	// 使用 map 去重，后加载的同名技能覆盖前面的
	skillsMap := make(map[string]domain.SkillInfo)

	for _, searchPath := range s.searchPaths {
		if !isDir(searchPath) {
			continue
		}

		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillName := entry.Name()
			skillFile := filepath.Join(searchPath, skillName, "SKILL.md")

			// 验证技能名称
			if err := validateSkillName(skillName); err != nil {
				continue
			}

			if _, err := os.Stat(skillFile); err != nil {
				continue
			}

			// 获取技能元数据
			meta := s.GetSkillMetadata(skillName)
			available := s.CheckRequirements(skillName)
			var requires string
			if !available {
				requires = s.GetMissingRequirements(skillName)
			}
			desc := ""
			if d, ok := meta["description"]; ok {
				desc = d
			}

			// 放入 map，后面的会覆盖前面的
			skillsMap[skillName] = domain.SkillInfo{
				Name:        skillName,
				Description: desc,
				Available:   available,
				Requires:    requires,
			}
		}
	}

	// 转换为切片
	skills := make([]domain.SkillInfo, 0, len(skillsMap))
	for _, info := range skillsMap {
		skills = append(skills, info)
	}

	return skills
}

// LoadSkill 加载技能内容（搜索所有路径）
func (s *SkillsLoader) LoadSkill(name string) string {
	// 验证技能名称，防止路径遍历攻击
	if err := validateSkillName(name); err != nil {
		return ""
	}

	// 从后往前搜索，后面的路径优先级更高
	for i := len(s.searchPaths) - 1; i >= 0; i-- {
		searchPath := s.searchPaths[i]
		skillFile := filepath.Join(searchPath, name, "SKILL.md")

		if data, err := os.ReadFile(skillFile); err == nil {
			return string(data)
		}
	}

	return ""
}

// LoadSkillContent 加载技能内容（不含 metadata）
func (s *SkillsLoader) LoadSkillContent(name string) string {
	content := s.LoadSkill(name)
	if content == "" {
		return ""
	}
	return s.stripFrontmatter(content)
}

// GetSkillMetadata 获取技能元数据
func (s *SkillsLoader) GetSkillMetadata(name string) map[string]string {
	// 验证技能名称
	if err := validateSkillName(name); err != nil {
		return nil
	}

	// 尝试从各个路径加载
	var content string
	for i := len(s.searchPaths) - 1; i >= 0; i-- {
		searchPath := s.searchPaths[i]
		skillFile := filepath.Join(searchPath, name, "SKILL.md")

		if data, err := os.ReadFile(skillFile); err == nil {
			content = string(data)
			break
		}
	}

	if content == "" {
		return nil
	}

	if !strings.HasPrefix(content, "---") {
		return nil
	}

	// 解析 YAML 前言
	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil
	}

	// 使用 yaml.v3 解析
	var metadata map[string]interface{}
	if err := yaml.Unmarshal([]byte(match[1]), &metadata); err != nil {
		return nil
	}

	result := make(map[string]string)
	for key, value := range metadata {
		if str, ok := value.(string); ok {
			result[key] = str
		}
	}

	return result
}

// CheckRequirements 检查技能需求是否满足
func (s *SkillsLoader) CheckRequirements(name string) bool {
	meta := s.GetSkillMetadata(name)
	if meta == nil {
		return true
	}

	// 检查二进制文件需求
	if bins, ok := meta["requires_bins"]; ok {
		for _, bin := range strings.Split(bins, ",") {
			bin = strings.TrimSpace(bin)
			if bin != "" && !hasBinary(bin) {
				return false
			}
		}
	}

	// 检查环境变量需求
	if envs, ok := meta["requires_env"]; ok {
		for _, env := range strings.Split(envs, ",") {
			env = strings.TrimSpace(env)
			if env != "" && config.GetEnv(env) == "" {
				return false
			}
		}
	}

	return true
}

// GetMissingRequirements 获取缺失的需求
func (s *SkillsLoader) GetMissingRequirements(name string) string {
	meta := s.GetSkillMetadata(name)
	if meta == nil {
		return ""
	}

	var missing []string

	if bins, ok := meta["requires_bins"]; ok {
		for _, bin := range strings.Split(bins, ",") {
			bin = strings.TrimSpace(bin)
			if bin != "" && !hasBinary(bin) {
				missing = append(missing, "CLI: "+bin)
			}
		}
	}

	if envs, ok := meta["requires_env"]; ok {
		for _, env := range strings.Split(envs, ",") {
			env = strings.TrimSpace(env)
			if env != "" && config.GetEnv(env) == "" {
				missing = append(missing, "ENV: "+env)
			}
		}
	}

	return strings.Join(missing, ", ")
}

// hasBinary 检查二进制文件是否存在
func hasBinary(name string) bool {
	path := config.GetPath()
	for _, dir := range strings.Split(path, string(os.PathListSeparator)) {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// stripFrontmatter 移除 YAML 前言
func (s *SkillsLoader) stripFrontmatter(content string) string {
	if strings.HasPrefix(content, "---") {
		re := regexp.MustCompile(`(?s)^---\n.*?\n---\n`)
		content = re.ReplaceAllString(content, "")
	}
	return strings.TrimSpace(content)
}

// validateSkillName 验证技能名称，防止路径遍历攻击
func validateSkillName(name string) error {
	// 检查是否包含路径分隔符
	if strings.Contains(name, string(filepath.Separator)) || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("skill name contains invalid path characters: %s", name)
	}

	// 检查是否包含父目录引用
	if strings.Contains(name, "..") {
		return fmt.Errorf("skill name contains parent directory reference: %s", name)
	}

	// 检查是否为空
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	return nil
}

// isDir 判断路径是否为目录
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
