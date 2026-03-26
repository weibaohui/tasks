/**
 * UseSkillTool - 技能动态加载工具
 * 用于按需加载可用技能列表
 */
package skill

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/skill"
)

// UseSkillTool use_skill 工具 - 用于按需加载技能
type UseSkillTool struct {
	skillsLoader *skill.SkillsLoader
}

// NewUseSkillTool 创建 use_skill 工具
func NewUseSkillTool(skillsLoader *skill.SkillsLoader) *UseSkillTool {
	return &UseSkillTool{
		skillsLoader: skillsLoader,
	}
}

var _ llm.Tool = (*UseSkillTool)(nil)

// Name 返回工具名称
func (t *UseSkillTool) Name() string {
	return "use_skill"
}

// Description 返回工具描述
func (t *UseSkillTool) Description() string {
	return `加载并使用指定技能。
调用此工具后，该技能将被激活可在当前对话中使用。
参数 skill_name: 技能名称（如 'skill-creator', 'code-review'）
参数 action: 操作类型，'load'(加载技能) 或 'info'(仅返回技能信息)`
}

// Parameters 返回参数 schema
func (t *UseSkillTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"skill_name": {
				"type": "string",
				"description": "技能名称（如 'skill-creator', 'code-review'）"
			},
			"action": {
				"type": "string",
				"description": "操作类型：'load'(加载技能), 'info'(仅返回技能信息)",
				"enum": ["load", "info"]
			}
		},
		"required": ["skill_name"]
	}`)
}

// Execute 执行工具
func (t *UseSkillTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	var args struct {
		SkillName string `json:"skill_name"`
		Action    string `json:"action"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("解析参数失败: %v", err),
		}, nil
	}

	if args.SkillName == "" {
		return &llm.ToolResult{
			Output: "",
			Error:  "skill_name 不能为空",
		}, nil
	}

	// 默认操作为 load
	if args.Action == "" {
		args.Action = "load"
	}

	switch args.Action {
	case "load":
		return t.handleLoad(args.SkillName)
	case "info":
		return t.handleInfo(args.SkillName)
	default:
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("不支持的操作: %s", args.Action),
		}, nil
	}
}

// handleLoad 加载技能
func (t *UseSkillTool) handleLoad(skillName string) (*llm.ToolResult, error) {
	if t.skillsLoader == nil {
		return &llm.ToolResult{
			Output: "",
			Error:  "技能加载器未配置",
		}, nil
	}

	// 获取所有技能
	skills := t.skillsLoader.ListSkills()

	// 查找指定技能
	var skillInfo *skill.SkillInfo
	for i, s := range skills {
		if s.Name == skillName {
			skillInfo = &skills[i]
			break
		}
	}

	if skillInfo == nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("技能 '%s' 不存在", skillName),
		}, nil
	}

	if !skillInfo.Available {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("技能 '%s' 不可用: %s", skillName, skillInfo.Requires),
		}, nil
	}

	// 加载技能内容
	content := t.skillsLoader.LoadSkillContent(skillName)
	if content == "" {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("技能 '%s' 内容加载失败", skillName),
		}, nil
	}

	result := map[string]interface{}{
		"success":    true,
		"skill_name": skillInfo.Name,
		"message":    fmt.Sprintf("技能 '%s' 加载成功", skillName),
		"content":    content,
		"usage":      "技能已激活，可直接使用",
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &llm.ToolResult{
		Output: string(resultJSON),
		Error:  "",
	}, nil
}

// handleInfo 获取技能信息
func (t *UseSkillTool) handleInfo(skillName string) (*llm.ToolResult, error) {
	if t.skillsLoader == nil {
		return &llm.ToolResult{
			Output: "",
			Error:  "技能加载器未配置",
		}, nil
	}

	// 获取所有技能
	skills := t.skillsLoader.ListSkills()

	// 查找指定技能
	var skillInfo *skill.SkillInfo
	for i, s := range skills {
		if s.Name == skillName {
			skillInfo = &skills[i]
			break
		}
	}

	if skillInfo == nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("技能 '%s' 不存在", skillName),
		}, nil
	}

	result := map[string]interface{}{
		"success":    true,
		"skill_name": skillInfo.Name,
		"description": skillInfo.Description,
		"available":  skillInfo.Available,
		"requires":   skillInfo.Requires,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &llm.ToolResult{
		Output: string(resultJSON),
		Error:  "",
	}, nil
}