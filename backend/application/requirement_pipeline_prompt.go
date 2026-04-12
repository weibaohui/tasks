package application

import (
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/domain/statemachine"
)

// buildExecutionPipeline 根据状态机配置动态生成执行流水线
// 使用 PHASE 结构和 ON_PHASE_COMPLETE Hook
func buildExecutionPipeline(smConfig *statemachine.Config, requirementID, gitRepo, branch string) string {
	if smConfig == nil || len(smConfig.Transitions) == 0 {
		return `PHASE 1 — EXECUTE
执行需求开发工作
ON_PHASE_COMPLETE:
taskmanager requirement transition --id ` + requirementID + ` --trigger=completed
`
	}

	// 构建状态链
	stateChain := buildStateChain(smConfig)
	if len(stateChain) == 0 {
		return `PHASE 1 — EXECUTE
执行需求开发工作
ON_PHASE_COMPLETE:
taskmanager requirement transition --id ` + requirementID + ` --trigger=completed
`
	}

	var sb strings.Builder
	phaseNum := 1

	// PHASE 0: 初始化环境（总是第一阶段）
	sb.WriteString(`PHASE 0 — INIT
执行：
`)
	sb.WriteString(fmt.Sprintf("git clone %s . && git checkout %s\n", gitRepo, branch))
	sb.WriteString(fmt.Sprintf("git checkout %s && git pull\n", branch))
	sb.WriteString(`ON_PHASE_COMPLETE:
`)
	sb.WriteString(fmt.Sprintf("taskmanager requirement transition --id %s --trigger=%s\n\n",
		requirementID, getFirstTrigger(smConfig)))

	// 遍历状态链生成 PHASE
	for i := 0; i < len(stateChain)-1; i++ {
		currentStateID := stateChain[i]
		nextStateID := stateChain[i+1]

		currentState := smConfig.GetState(currentStateID)
		if currentState == nil {
			continue
		}

		// 跳过终态
		if currentState.IsFinal {
			continue
		}

		// 查找转换获取触发器
		trigger := ""
		for _, t := range smConfig.Transitions {
			if t.FromState == currentStateID && t.ToState == nextStateID {
				trigger = t.Trigger
				break
			}
		}

		// PHASE 头部
		phaseName := strings.ToUpper(strings.ReplaceAll(currentState.Name, " ", "_"))
		sb.WriteString(fmt.Sprintf("PHASE %d — %s\n", phaseNum, phaseName))

		// 添加 AI 指南中的执行指令（提取命令，排除状态转换命令）
		aiGuide := strings.TrimSpace(currentState.AIGuide)
		if aiGuide != "" {
			guideLines := strings.Split(aiGuide, "\n")
			for _, line := range guideLines {
				trimmed := strings.TrimSpace(line)
				// 跳过包含"命令："、taskmanager、## 标题、**标记的行
				if strings.Contains(trimmed, "命令：") ||
					strings.HasPrefix(trimmed, "taskmanager") ||
					strings.HasPrefix(trimmed, "## ") ||
					strings.HasPrefix(trimmed, "**") ||
					trimmed == "" {
					continue
				}
				sb.WriteString(trimmed + "\n")
			}
		}

		// 如果是第一个业务 PHASE，添加环境初始化提示
		if phaseNum == 1 {
			sb.WriteString("\n执行环境初始化（已在 PHASE 0 完成）\n")
		}

		// ON_PHASE_COMPLETE Hook
		if trigger != "" {
			sb.WriteString("\nON_PHASE_COMPLETE:\n")
			sb.WriteString(fmt.Sprintf("taskmanager requirement transition --id %s --trigger=%s\n",
				requirementID, trigger))
		}

		sb.WriteString("\n")
		phaseNum++
	}

	// 如果没有生成任何 PHASE
	if phaseNum == 1 {
		return `PHASE 1 — EXECUTE
执行需求开发工作
ON_PHASE_COMPLETE:
taskmanager requirement transition --id ` + requirementID + ` --trigger=completed
`
	}

	return sb.String()
}

// getFirstTrigger 获取状态机的第一个触发器（从 initial_state 出发）
func getFirstTrigger(smConfig *statemachine.Config) string {
	if smConfig == nil {
		return "start"
	}
	for _, t := range smConfig.Transitions {
		if t.FromState == smConfig.InitialState {
			return t.Trigger
		}
	}
	return "start"
}


// buildStateChain 从状态机构建状态链
// 根据 transitions 构建从 initial_state 开始的完整状态路径
func buildStateChain(smConfig *statemachine.Config) []string {
	if smConfig == nil {
		return nil
	}

	// 构建转换映射：from_state -> to_state
	transitionMap := make(map[string]string)
	for _, t := range smConfig.Transitions {
		transitionMap[t.FromState] = t.ToState
	}

	// 从 initial_state 开始，沿着转换链构建状态数组
	var chain []string
	currentState := smConfig.InitialState

	for i := 0; i < len(smConfig.States)+1; i++ { // 防止无限循环
		chain = append(chain, currentState)
		nextState, exists := transitionMap[currentState]
		if !exists {
			break
		}
		// 防止循环
		if contains(chain, nextState) {
			break
		}
		currentState = nextState
	}

	return chain
}
