/**
 * Skill Tool Adapter
 * 将 skill.DynamicTool 适配到 llm.Tool 接口
 */
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/weibh/taskmanager/infrastructure/llm"
	skilltool "github.com/weibh/taskmanager/infrastructure/llm/tools/skill"
	"github.com/weibh/taskmanager/infrastructure/skill"
)

// SkillToolAdapter 技能工具适配器，实现 llm.Tool 接口
type SkillToolAdapter struct {
	skillTool *skilltool.DynamicTool
}

// NewSkillToolAdapter 创建技能工具适配器
func NewSkillToolAdapter(skillTool *skilltool.DynamicTool) *SkillToolAdapter {
	return &SkillToolAdapter{
		skillTool: skillTool,
	}
}

// 确保实现 llm.Tool 接口
var _ llm.Tool = (*SkillToolAdapter)(nil)

// Name 返回工具名称
func (a *SkillToolAdapter) Name() string {
	return "skill__" + a.skillTool.Name()
}

// Description 返回工具描述
func (a *SkillToolAdapter) Description() string {
	desc := a.skillTool.Description()
	if desc == "" {
		desc = fmt.Sprintf("技能工具：%s。使用此工具可以加载并执行 %s 技能的相关操作。", a.skillTool.Name(), a.skillTool.Name())
	}
	return desc
}

// Parameters 返回参数 schema
func (a *SkillToolAdapter) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"description": "要执行的操作（可选），由技能定义"
			},
			"params": {
				"type": "object",
				"description": "操作参数（可选）"
			}
		}
	}`)
}

// Execute 执行工具
func (a *SkillToolAdapter) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	// 执行技能
	result, err := a.skillTool.InvokableRun(ctx, string(input))
	if err != nil {
		return &llm.ToolResult{
			Error: err.Error(),
		}, nil
	}

	return &llm.ToolResult{
		Output: result,
	}, nil
}

// SkillToolsAdapterRegistry 技能工具注册器
type SkillToolsAdapterRegistry struct {
	loader *skill.SkillsLoader
}

// NewSkillToolsAdapterRegistry 创建技能工具适配器注册器
func NewSkillToolsAdapterRegistry(loader *skill.SkillsLoader) *SkillToolsAdapterRegistry {
	return &SkillToolsAdapterRegistry{
		loader: loader,
	}
}

// GetTools 返回所有适配后的技能工具
func (r *SkillToolsAdapterRegistry) GetTools() []llm.Tool {
	if r.loader == nil {
		return nil
	}

	skills := r.loader.ListSkills()
	tools := make([]llm.Tool, 0, len(skills))

	for _, s := range skills {
		// 检查技能是否可用
		if !s.Available {
			continue
		}

		// 创建动态工具
		dynamicTool := skilltool.NewDynamicTool(s.Name, s.Description, r.loader.LoadSkill)
		// 创建适配器
		adapter := NewSkillToolAdapter(dynamicTool)
		tools = append(tools, adapter)
	}

	return tools
}
