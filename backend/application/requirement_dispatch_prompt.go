package application

import (
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
)

func buildRequirementDispatchPrompt(requirement *domain.Requirement, project *domain.Project, workspacePath string, stateMachineName string, currentState string, aiGuide map[string]interface{}, smConfig *statemachine.Config) string {
	isHeartbeat := requirement.RequirementType() == domain.RequirementTypeHeartbeat
	requirementType := "普通需求"
	if isHeartbeat {
		requirementType = "心跳需求"
	}

	// 构建状态机使用指南和 AI 指南
	stateMachineGuide := buildStateMachineGuide(stateMachineName, smConfig)
	stateAIGuide := buildStateAIGuide(aiGuide)

	// 心跳需求：调度员角色，不直接修改代码
	var prompt string
	if isHeartbeat {
		prompt = fmt.Sprintf(`你是当前项目的调度员心跳（Heartbeat Agent），职责是 orchestrate 任务而非直接修改代码。

【需求元信息】
- 需求ID：%s
- 需求类型：%s
- 当前状态：%s
- 状态机当前状态：%s
- 需求标题：%s
- 需求描述：%s
- 项目ID：%s
- 项目名称：%s
- 关联状态机：%s

【验收标准】
%s

【项目信息】
- 仓库地址：%s
- 默认分支：%s

【工作目录】
- 当前工作目录：%s

%s

%s

【执行流程 - 心跳任务】
你是调度员，负责编排任务而非直接修改代码。请按照以下心跳任务定义执行：

%s

【调度员核心守则】
1. **严禁**直接修改任何源代码文件
2. **严禁**执行 git commit、git push、创建 PR
3. 所有代码改动必须通过 "taskmanager requirement create" 创建新需求，交由 CodingAgent 完成
4. 根据工作结果，使用状态机命令更新需求状态
5. 工作完成后，输出本次心跳的执行结果摘要
`, requirement.ID().String(), requirementType, requirement.Status(), firstNonEmpty(currentState, "未初始化"), requirement.Title(), firstNonEmpty(requirement.Description(), "无"),
			project.ID().String(), project.Name(), firstNonEmpty(stateMachineName, "未配置"),
			firstNonEmpty(requirement.AcceptanceCriteria(), "完成调度工作"),
			project.GitRepoURL(), project.DefaultBranch(), workspacePath,
			stateAIGuide,
			stateMachineGuide,
			project.HeartbeatMDContent())
	} else {
		initSteps := project.InitSteps()
		initStepsText := "无"
		if len(initSteps) > 0 {
			initStepsText = strings.Join(initSteps, "\n")
		}
		prompt = fmt.Sprintf(`你是当前需求的 CodingAgent 分身，请直接使用 Claude Code 在当前工作目录完成开发。

【执行契约 - 严格遵守】
当前阶段：%s
下一步动作：执行状态转换进入下一阶段

执行规则：
1. 必须执行命令，不输出解释性长文本
2. 每个 PHASE 完成后必须执行 ON_PHASE_COMPLETE 命令
3. 命令失败自动修复重试（最多3次）
4. 优先执行操作，不做过多分析

【需求元信息】
- 需求ID：%s
- 需求类型：%s
- 需求标题：%s
- 需求描述：%s
- 项目ID：%s
- 项目名称：%s

【验收标准】
%s

【项目信息】
- 仓库地址：%s
- 默认分支：%s
- 初始化步骤：
%s

【工作目录】
- 当前工作目录：%s
- 要求：必须在该目录内完成代码操作

%s

【工作循环 - 持续执行】
1. 规划当前阶段任务
2. 执行命令
3. 验证执行结果
4. 若失败 → 自动修复 → 重试
5. 进入下一 PHASE

【禁止行为】
- 修改与需求无关的文件
- 跳过测试或验证步骤
- 提交未通过验证的代码
- 创建额外分支

【失败处理】
若命令失败：
1. 分析错误输出
2. 自动修复
3. 重试最多3次
4. 仍失败则输出阻塞原因并停止


【执行流水线 - PIPELINE】
%s

【状态机附录 - 调试参考】
%s
`, firstNonEmpty(currentState, "未初始化"),
			requirement.ID().String(), requirementType, requirement.Title(), firstNonEmpty(requirement.Description(), "无"),
			project.ID().String(), project.Name(),
			firstNonEmpty(requirement.AcceptanceCriteria(), "完成需求并通过验证"),
			project.GitRepoURL(), project.DefaultBranch(), initStepsText, workspacePath,
			stateAIGuide,
			buildExecutionPipeline(smConfig, requirement.ID().String(), project.GitRepoURL(), project.DefaultBranch()),
			stateMachineGuide)
	}
	return prompt
}

// getStateName 根据状态ID获取状态名称
func getStateName(smConfig *statemachine.Config, stateID string) string {
	for _, s := range smConfig.States {
		if s.ID == stateID {
			return s.Name
		}
	}
	return stateID
}


