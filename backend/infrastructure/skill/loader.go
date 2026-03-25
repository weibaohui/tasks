/**
 * Skills Loader
 * 技能加载器 - 从文件系统加载技能
 */
package skill

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SkillLoaderFunc 技能加载函数类型
type SkillLoaderFunc func(name string) string

// SkillInfo 技能信息
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // "workspace" 或 "builtin"
	Path        string `json:"path"`
	Available   bool   `json:"available"`
	Requires    string `json:"requires,omitempty"`
}

// SkillsLoader 技能加载器
type SkillsLoader struct {
	workspace       string
	workspaceSkills string
	builtinSkills   string
}

// NewSkillsLoader 创建技能加载器
func NewSkillsLoader(workspace string) *SkillsLoader {
	return &SkillsLoader{
		workspace:       workspace,
		workspaceSkills: filepath.Join(workspace, "skills"),
		builtinSkills:   detectBuiltinSkillsDir(),
	}
}

// GetWorkspaceSkills 获取工作区技能目录路径
func (s *SkillsLoader) GetWorkspaceSkills() string {
	return s.workspaceSkills
}

// ListSkills 列出所有可用技能
func (s *SkillsLoader) ListSkills() []SkillInfo {
	var skills []SkillInfo

	// 工作区技能（最高优先级）
	if dir, err := os.ReadDir(s.workspaceSkills); err == nil {
		for _, entry := range dir {
			if entry.IsDir() {
				skillFile := filepath.Join(s.workspaceSkills, entry.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					meta := s.GetSkillMetadata(entry.Name())
					available := s.CheckRequirements(entry.Name())
					var requires string
					if !available {
						requires = s.GetMissingRequirements(entry.Name())
					}
					desc := ""
					if d, ok := meta["description"]; ok {
						desc = d
					}
					skills = append(skills, SkillInfo{
						Name:        entry.Name(),
						Description: desc,
						Source:      "workspace",
						Path:        skillFile,
						Available:   available,
						Requires:    requires,
					})
				}
			}
		}
	}

	// 内置技能
	if s.builtinSkills != "" {
		if dir, err := os.ReadDir(s.builtinSkills); err == nil {
			for _, entry := range dir {
				if entry.IsDir() {
					skillFile := filepath.Join(s.builtinSkills, entry.Name(), "SKILL.md")
					if _, err := os.Stat(skillFile); err == nil {
						// 检查是否已存在
						exists := false
						for _, sk := range skills {
							if sk.Name == entry.Name() {
								exists = true
								break
							}
						}
						if !exists {
							meta := s.GetSkillMetadata(entry.Name())
							available := s.CheckRequirements(entry.Name())
							var requires string
							if !available {
								requires = s.GetMissingRequirements(entry.Name())
							}
							desc := ""
							if d, ok := meta["description"]; ok {
								desc = d
							}
							skills = append(skills, SkillInfo{
								Name:        entry.Name(),
								Description: desc,
								Source:      "builtin",
								Path:        skillFile,
								Available:   available,
								Requires:    requires,
							})
						}
					}
				}
			}
		}
	}

	return skills
}

// LoadSkill 加载技能内容
func (s *SkillsLoader) LoadSkill(name string) string {
	// 先检查工作区
	workspaceSkill := filepath.Join(s.workspaceSkills, name, "SKILL.md")
	if data, err := os.ReadFile(workspaceSkill); err == nil {
		return string(data)
	}

	// 检查内置
	if s.builtinSkills != "" {
		builtinSkill := filepath.Join(s.builtinSkills, name, "SKILL.md")
		if data, err := os.ReadFile(builtinSkill); err == nil {
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
	content := s.LoadSkill(name)
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

	metadata := make(map[string]string)
	for _, line := range strings.Split(match[1], "\n") {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"'")
			metadata[key] = value
		}
	}

	return metadata
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
			if bin != "" && !s.hasBinary(bin) {
				return false
			}
		}
	}

	// 检查环境变量需求
	if envs, ok := meta["requires_env"]; ok {
		for _, env := range strings.Split(envs, ",") {
			env = strings.TrimSpace(env)
			if env != "" && os.Getenv(env) == "" {
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
			if bin != "" && !s.hasBinary(bin) {
				missing = append(missing, "CLI: "+bin)
			}
		}
	}

	if envs, ok := meta["requires_env"]; ok {
		for _, env := range strings.Split(envs, ",") {
			env = strings.TrimSpace(env)
			if env != "" && os.Getenv(env) == "" {
				missing = append(missing, "ENV: "+env)
			}
		}
	}

	return strings.Join(missing, ", ")
}

// hasBinary 检查二进制文件是否存在
func (s *SkillsLoader) hasBinary(name string) bool {
	path := os.Getenv("PATH")
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

// detectBuiltinSkillsDir 解析内置技能目录
func detectBuiltinSkillsDir() string {
	// 优先从环境变量读取
	if env := strings.TrimSpace(os.Getenv("TASKMANAGER_SKILLS_DIR")); env != "" {
		return env
	}

	// 从当前工作目录查找
	if cwd, err := os.Getwd(); err == nil {
		dir := filepath.Join(cwd, "skills")
		if isDir(dir) {
			return dir
		}
	}

	// 从可执行文件目录查找
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "skills")
		if isDir(dir) {
			return dir
		}
	}

	return ""
}

// isDir 判断路径是否为目录
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}