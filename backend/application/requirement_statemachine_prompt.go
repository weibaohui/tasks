package application

import (
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/domain/statemachine"
)

// buildStateMachineGuide 构建状态机使用指南
func buildStateMachineGuide(stateMachineName string, smConfig *statemachine.Config) string {
	if stateMachineName == "" {
		return `【状态机使用指南】
当前需求未配置状态机。如需使用状态机管理工作流，请联系管理员配置。`
	}

	guide := fmt.Sprintf(`【状态机使用指南】
本需求关联的状态机名称：%s

你可以使用以下命令管理需求状态：

【推荐】一键状态转换（自动同步状态机+需求状态）：
   taskmanager requirement transition --id <需求ID> --trigger=<触发器>

【分步操作】如需更细粒度控制：

1. 查看当前需求状态：
   taskmanager requirement get-state --id <需求ID>

2. 查看当前状态可用触发器：
   taskmanager statemachine triggers --machine=<状态机名称> --from=<当前状态>

3. 验证状态转换是否合法：
   taskmanager statemachine validate --machine=<状态机名称> --from=<当前状态> --to=<目标状态>

4. 执行状态转换并同步需求状态：
   taskmanager requirement transition --id <需求ID> --trigger=<触发器>
`,
		stateMachineName)

	// 如果有状态机配置，注入完整触发器表
	if smConfig != nil {
		triggerTable := buildTriggerTable(smConfig)
		if triggerTable != "" {
			guide += fmt.Sprintf(`【状态机触发器表】
%s

`, triggerTable)
		}
	}

	return guide
}

// buildTriggerTable 构建状态机的完整触发器表（Markdown格式）
// 包含所有状态和从该状态出发可用的所有触发器
func buildTriggerTable(smConfig *statemachine.Config) string {
	if smConfig == nil || len(smConfig.States) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("| 源状态 | 触发器 | 目标状态 | 说明 |\n")
	sb.WriteString("|--------|--------|----------|------|\n")

	// 按状态分组，显示每个状态可用的触发器
	for _, state := range smConfig.States {
		if state.IsFinal {
			// 终态不显示触发器（不会有从终态出发的转换）
			continue
		}

		triggers := smConfig.GetAvailableTriggers(state.ID)
		if len(triggers) == 0 {
			// 没有触发器的状态显示为 -
			sb.WriteString(fmt.Sprintf("| %s | - | - | 无可用触发器 |\n", state.Name))
			continue
		}

		// 每个触发器一行
		for i, trigger := range triggers {
			// 查找对应的转换获取目标状态
			transition := findTransitionByTrigger(smConfig, state.ID, trigger.Trigger)
			toState := ""
			description := trigger.Description
			if transition != nil {
				toState = getStateName(smConfig, transition.ToState)
			}

			// 第一行显示状态名，后续行留空
			fromState := state.Name
			if i > 0 {
				fromState = ""
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				fromState, trigger.Trigger, toState, description))
		}
	}

	return sb.String()
}

// findTransitionByTrigger 根据触发器查找转换
func findTransitionByTrigger(smConfig *statemachine.Config, fromState, trigger string) *statemachine.Transition {
	for i := range smConfig.Transitions {
		t := &smConfig.Transitions[i]
		if t.FromState == fromState && t.Trigger == trigger {
			return t
		}
	}
	return nil
}


// buildStateAIGuide 构建状态的 AI 指南
func buildStateAIGuide(aiGuide map[string]interface{}) string {
	if aiGuide == nil {
		return "【当前状态执行指南】\n暂无详细的 AI 执行指南。请根据需求描述和验收标准自行判断当前阶段的工作内容。"
	}

	var result strings.Builder
	result.WriteString("【当前状态执行指南】\n")

	// 状态名称
	if name, ok := aiGuide["name"].(string); ok && name != "" {
		result.WriteString(fmt.Sprintf("当前阶段：%s\n\n", name))
	}

	// AI Guide
	if guide, ok := aiGuide["ai_guide"].(string); ok && guide != "" {
		result.WriteString(guide)
		result.WriteString("\n\n")
	}

	// 自动初始化
	if autoInit, ok := aiGuide["auto_init"].(string); ok && autoInit != "" {
		result.WriteString("【自动初始化命令】\n")
		result.WriteString("进入此状态时，将自动执行以下命令：\n")
		result.WriteString("```bash\n")
		result.WriteString(autoInit)
		result.WriteString("\n```\n\n")
	}

	// 成功判断标准
	if success, ok := aiGuide["success_criteria"].(string); ok && success != "" {
		result.WriteString("【成功判断标准】\n")
		result.WriteString(success)
		result.WriteString("\n\n")
	}

	// 失败判断标准
	if failure, ok := aiGuide["failure_criteria"].(string); ok && failure != "" {
		result.WriteString("【失败判断标准】\n")
		result.WriteString(failure)
		result.WriteString("\n\n")
	}

	// 可用触发器
	if triggers, ok := aiGuide["triggers"].([]interface{}); ok && len(triggers) > 0 {
		result.WriteString("【可用状态转换】\n")
		result.WriteString("根据工作结果，选择合适的触发器执行转换：\n\n")
		for _, t := range triggers {
			if trigger, ok := t.(map[string]interface{}); ok {
				triggerName := ""
				description := ""
				condition := ""

				if n, ok := trigger["trigger"].(string); ok {
					triggerName = n
				}
				if d, ok := trigger["description"].(string); ok {
					description = d
				}
				if c, ok := trigger["condition"].(string); ok {
					condition = c
				}

				result.WriteString(fmt.Sprintf("• %s", triggerName))
				if description != "" {
					result.WriteString(fmt.Sprintf(" - %s", description))
				}
				result.WriteString("\n")
				if condition != "" {
					result.WriteString(fmt.Sprintf("  触发条件：%s\n", condition))
				}
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
