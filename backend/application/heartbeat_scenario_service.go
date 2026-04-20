package application

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

type HeartbeatScenarioService struct {
	scenarioRepo  domain.HeartbeatScenarioRepository
	projectRepo   domain.ProjectRepository
	heartbeatRepo domain.HeartbeatRepository
	bindingRepo   domain.WebhookHeartbeatBindingRepository
	idGenerator   domain.IDGenerator
	scheduler     *HeartbeatScheduler
}

// HeartbeatApplyPreview 表示将场景应用到项目前的影响预览结果。
type HeartbeatApplyPreview struct {
	ProjectID       string
	ProjectName     string
	ScenarioCode    string
	ScenarioName    string
	CurrentScenario string
	ToDelete        []*domain.Heartbeat
	ToCreate        []*domain.Heartbeat
}

func NewHeartbeatScenarioService(
	scenarioRepo domain.HeartbeatScenarioRepository,
	projectRepo domain.ProjectRepository,
	heartbeatRepo domain.HeartbeatRepository,
	bindingRepo domain.WebhookHeartbeatBindingRepository,
	idGenerator domain.IDGenerator,
	scheduler *HeartbeatScheduler,
) *HeartbeatScenarioService {
	return &HeartbeatScenarioService{
		scenarioRepo:  scenarioRepo,
		projectRepo:   projectRepo,
		heartbeatRepo: heartbeatRepo,
		bindingRepo:   bindingRepo,
		idGenerator:   idGenerator,
		scheduler:     scheduler,
	}
}

func (s *HeartbeatScenarioService) CreateScenario(ctx context.Context, code, name, description string, items []domain.HeartbeatScenarioItem, enabled bool) (*domain.HeartbeatScenario, error) {
	scenario, err := domain.NewHeartbeatScenario(
		domain.NewHeartbeatScenarioID(s.idGenerator.Generate()),
		code,
		name,
		description,
		items,
	)
	if err != nil {
		return nil, err
	}
	scenario.SetEnabled(enabled)
	if err := s.scenarioRepo.Save(ctx, scenario); err != nil {
		return nil, fmt.Errorf("failed to save scenario: %w", err)
	}
	return scenario, nil
}

func (s *HeartbeatScenarioService) GetScenarioByCode(ctx context.Context, code string) (*domain.HeartbeatScenario, error) {
	return s.scenarioRepo.FindByCode(ctx, code)
}

func (s *HeartbeatScenarioService) ListScenarios(ctx context.Context) ([]*domain.HeartbeatScenario, error) {
	return s.scenarioRepo.FindAll(ctx)
}

func (s *HeartbeatScenarioService) UpdateScenario(ctx context.Context, code, name, description string, items []domain.HeartbeatScenarioItem, enabled bool) (*domain.HeartbeatScenario, error) {
	scenario, err := s.scenarioRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to find scenario: %w", err)
	}
	if scenario == nil {
		return nil, fmt.Errorf("scenario not found")
	}
	if err := scenario.Update(name, description, items); err != nil {
		return nil, err
	}
	scenario.SetEnabled(enabled)
	if err := s.scenarioRepo.Save(ctx, scenario); err != nil {
		return nil, fmt.Errorf("failed to save scenario: %w", err)
	}
	return scenario, nil
}

func (s *HeartbeatScenarioService) DeleteScenario(ctx context.Context, id string) error {
	scenario, err := s.scenarioRepo.FindByID(ctx, domain.NewHeartbeatScenarioID(id))
	if err != nil {
		return err
	}
	if scenario == nil {
		return fmt.Errorf("scenario not found")
	}
	if scenario.IsBuiltIn() {
		return fmt.Errorf("cannot delete built-in scenario")
	}
	return s.scenarioRepo.Delete(ctx, domain.NewHeartbeatScenarioID(id))
}

// PreviewApplyScenarioToProject 预览项目应用场景后的心跳增删影响，不会写入数据库。
func (s *HeartbeatScenarioService) PreviewApplyScenarioToProject(ctx context.Context, projectID, scenarioCode string) (*HeartbeatApplyPreview, error) {
	project, err := s.projectRepo.FindByID(ctx, domain.NewProjectID(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to find project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	scenario, err := s.scenarioRepo.FindByCode(ctx, scenarioCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find scenario: %w", err)
	}
	if scenario == nil {
		return nil, fmt.Errorf("scenario not found")
	}

	existingHeartbeats, err := s.heartbeatRepo.FindByProjectID(ctx, domain.NewProjectID(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to list existing heartbeats: %w", err)
	}

	toCreate, err := scenario.ApplyToProject(project.ID(), s.idGenerator, project.PlatformType())
	if err != nil {
		return nil, fmt.Errorf("failed to apply scenario: %w", err)
	}

	return &HeartbeatApplyPreview{
		ProjectID:       project.ID().String(),
		ProjectName:     project.Name(),
		ScenarioCode:    scenario.Code(),
		ScenarioName:    scenario.Name(),
		CurrentScenario: project.HeartbeatScenarioCode(),
		ToDelete:        existingHeartbeats,
		ToCreate:        toCreate,
	}, nil
}

// ApplyScenarioToProject 为项目应用场景，创建该场景下的所有心跳
func (s *HeartbeatScenarioService) ApplyScenarioToProject(ctx context.Context, projectID, scenarioCode string) error {
	project, err := s.projectRepo.FindByID(ctx, domain.NewProjectID(projectID))
	if err != nil {
		return fmt.Errorf("failed to find project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("project not found")
	}

	scenario, err := s.scenarioRepo.FindByCode(ctx, scenarioCode)
	if err != nil {
		return fmt.Errorf("failed to find scenario: %w", err)
	}
	if scenario == nil {
		return fmt.Errorf("scenario not found")
	}

	// 简化策略：删除该项目下所有现有心跳（由场景产生的或未修改的）
	// 然后重新创建。未来可以优化为只删除由同一场景生成的心跳。
	existingHeartbeats, err := s.heartbeatRepo.FindByProjectID(ctx, domain.NewProjectID(projectID))
	if err != nil {
		return fmt.Errorf("failed to list existing heartbeats: %w", err)
	}
	for _, hb := range existingHeartbeats {
		// 注意：不删除 bindings，因为 bindings 可能是用户手动创建的。
		// 即使心跳被删除，binding 会在触发时发现心跳不存在而被跳过。
		if err := s.heartbeatRepo.Delete(ctx, hb.ID()); err != nil {
			return fmt.Errorf("failed to delete existing heartbeat: %w", err)
		}
		if s.scheduler != nil {
			if err := s.scheduler.RefreshSchedule(ctx, hb.ID().String()); err != nil {
				return fmt.Errorf("failed to refresh schedule: %w", err)
			}
		}
	}

	// 实例化场景心跳
	heartbeats, err := scenario.ApplyToProject(project.ID(), s.idGenerator, project.PlatformType())
	if err != nil {
		return fmt.Errorf("failed to apply scenario: %w", err)
	}

	for _, hb := range heartbeats {
		if err := s.heartbeatRepo.Save(ctx, hb); err != nil {
			return fmt.Errorf("failed to save heartbeat: %w", err)
		}
		if s.scheduler != nil {
			if err := s.scheduler.RefreshSchedule(ctx, hb.ID().String()); err != nil {
				return fmt.Errorf("failed to refresh schedule: %w", err)
			}
		}
	}

	// 更新项目的场景编码
	project.SetHeartbeatScenarioCode(scenarioCode)
	if err := s.projectRepo.Save(ctx, project); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// EnsureBuiltInScenarios 确保内置场景存在，并更新已有内置场景的定义
func (s *HeartbeatScenarioService) EnsureBuiltInScenarios(ctx context.Context) error {
	// 确保 GitHub 开发协作工作流场景存在
	if err := s.ensureOneBuiltInScenario(ctx, BuildGitHubDevWorkflowScenario); err != nil {
		return err
	}

	// 确保 AMC 开发协作工作流场景存在
	if err := s.ensureOneBuiltInScenario(ctx, BuildAMCDevWorkflowScenario); err != nil {
		return err
	}

	return nil
}

// ensureOneBuiltInScenario 确保单个内置场景存在
func (s *HeartbeatScenarioService) ensureOneBuiltInScenario(ctx context.Context, builder func(string) *domain.HeartbeatScenario) error {
	expected := builder(s.idGenerator.Generate())

	existing, err := s.scenarioRepo.FindByCode(ctx, expected.Code())
	if err != nil {
		return fmt.Errorf("failed to check built-in scenario: %w", err)
	}
	scenarioContentChanged := false
	if existing != nil {
		// 检查内容是否有变化（name、description、items）
		nameChanged := existing.Name() != expected.Name()
		descChanged := existing.Description() != expected.Description()
		itemsChanged := !scenarioItemsEqual(existing.Items(), expected.Items())

		if nameChanged || descChanged || itemsChanged {
			// 更新现有内置场景的定义（保留 ID、创建时间、启用状态）
			existing.Update(expected.Name(), expected.Description(), expected.Items())
			existing.SetIsBuiltIn(true)
			if err := s.scenarioRepo.Save(ctx, existing); err != nil {
				return fmt.Errorf("failed to update built-in scenario: %w", err)
			}
			scenarioContentChanged = true
		}
	} else {
		if err := s.scenarioRepo.Save(ctx, expected); err != nil {
			return fmt.Errorf("failed to save built-in scenario: %w", err)
		}
	}

	// 若内置场景内容有变化，同步刷新所有已绑定该场景的项目的具体心跳实例
	if scenarioContentChanged {
		projects, err := s.projectRepo.FindAll(ctx)
		if err != nil {
			return fmt.Errorf("failed to list projects for scenario refresh: %w", err)
		}
		for _, project := range projects {
			if project.HeartbeatScenarioCode() == expected.Code() {
				if err := s.ApplyScenarioToProject(ctx, project.ID().String(), expected.Code()); err != nil {
					// 记录但不阻断启动流程
					fmt.Printf("[EnsureBuiltInScenarios] failed to refresh scenario for project %s: %v\n", project.Name(), err)
				}
			}
		}
	}

	return nil
}

// BuildGitHubDevWorkflowScenario 构建 GitHub 开发协作内置场景
func BuildGitHubDevWorkflowScenario(id string) *domain.HeartbeatScenario {
	items := []domain.HeartbeatScenarioItem{
		{
			Name:            "Issue 分析",
			IntervalMinutes: 180,
			RequirementType: "github_issue",
			AgentCode:       "",
			SortOrder:       1,
			MDContent: "你是项目的自动化协作助手。当前任务是：分析项目中的 open issues。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 gh issue list --repo owner/repo --state open 获取 open issues。\n" +
				"2. 对每个 issue，检查最近 3 小时内是否已有你的评论。如有，跳过该 issue（冷却机制）。\n" +
				"3. 对未被跳过的 issue，clone 仓库到临时目录，结合代码库分析 issue 描述。\n" +
				"4. 将分析结论（问题根因、可能影响的文件、建议修复方向）以评论形式发布到该 issue 下。\n" +
				"5. 如果没有 open issues，或全部处于冷却期，直接返回\"当前无待分析 issue\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 不修改任何源代码。\n" +
				"- 每次最多分析 2 个 issue。",
		},
		{
			Name:            "LGTM 代码编写",
			IntervalMinutes: 120,
			RequirementType: "github_coding",
			AgentCode:       "",
			SortOrder:       2,
			MDContent: "你是项目的自动化协作助手。当前任务是：为已评审通过的 issue 编写代码并创建 PR。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 gh issue list --repo owner/repo --label lgtm --state open 获取已评审通过的 issue。\n" +
				"2. 检查最旧的一个 issue 最近 3 小时内是否已有你的评论或关联 PR。如有，跳过。\n" +
				"3. 对选中的 issue：clone 仓库、创建 feature 分支、根据 issue 描述实现代码修改。\n" +
				"4. 如有测试命令，运行基础测试确保通过。\n" +
				"5. push 分支后使用 gh pr create 创建 PR，并在描述中通过 Closes #issue_number 关联 issue。\n" +
				"6. 如果没有带 lgtm 标签的 open issue，或全部处于冷却期，直接返回\"当前无待编写代码的 issue\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 每次心跳只处理一个 issue。\n" +
				"- 如遇到编译/测试失败，在 PR 或 issue 下评论说明，不强制提交。",
		},
		{
			Name:            "PR 需求评审",
			IntervalMinutes: 120,
			RequirementType: "github_pr_review",
			AgentCode:       "",
			SortOrder:       3,
			MDContent: "你是项目的自动化协作助手。当前任务是：检查 open PR 是否已完成需求评审。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 gh pr list --repo owner/repo --state open 获取 open PRs。\n" +
				"2. 对每个 PR，先检查最近 3 小时内是否已有你的评论。如有，跳过。\n" +
				"3. 再检查评论中是否包含\"需求评审通过\"字样。若已存在，跳过。\n" +
				"4. 若不存在，通过 PR 描述或 API 查找关联的 issue，阅读 issue 内容后在 PR 下评论：\n" +
				"   - 原始需求摘要\n" +
				"   - 需求评审结论（通过/需补充信息）\n" +
				"5. 如果所有 PR 都已通过需求评审或处于冷却期，直接返回\"所有 PR 已完成需求评审\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 不修改任何源代码。\n" +
				"- 每次最多评审 2 个 PR。",
		},
		{
			Name:            "PR 代码质量评审",
			IntervalMinutes: 180,
			RequirementType: "github_pr_review",
			AgentCode:       "",
			SortOrder:       4,
			MDContent: "你是项目的自动化协作助手。当前任务是：对已通过需求评审的 open PR 进行代码质量和安全评审。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 gh pr list --repo owner/repo --state open 获取 open PRs。\n" +
				"2. 对每个 PR，检查是否已有\"需求评审通过\"评论。若无，跳过该 PR（阶段闸门）。\n" +
				"3. 检查该 PR 最近 3 小时内是否已有你的评论。如有，跳过（冷却机制）。\n" +
				"4. 使用 gh pr view 和 gh pr diff 查看变更，从代码质量、潜在 bug、安全漏洞、性能问题等角度进行评审。\n" +
				"5. 将评审意见以评论形式发布到 PR 下。\n" +
				"6. 如果没有符合条件的 PR，直接返回\"当前无待评审 PR\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 不修改任何源代码。\n" +
				"- 每次最多评审 2 个 PR。",
		},
		{
			Name:            "PR 修改修复",
			IntervalMinutes: 240,
			RequirementType: "github_coding",
			AgentCode:       "",
			SortOrder:       5,
			MDContent: "你是项目的自动化协作助手。当前任务是：修复 PR 中的修改建议。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 查找当前项目下由你（AI 助手）创建或关联的 open PR。\n" +
				"2. 检查每个 PR 是否有未解决的评审建议。若无，跳过。\n" +
				"3. 检查最近 3 小时内你是否已对该 PR 评论或 push。如有，跳过（冷却机制，给人类留出确认时间）。\n" +
				"4. 阅读 PR 评论中的修改建议，判断哪些是可执行的合理建议。\n" +
				"5. checkout PR 分支、按建议修改代码、commit、push 更新。\n" +
				"6. 在 PR 下评论说明已修复的内容。\n" +
				"7. 如果没有待修复建议或全部处于冷却期，直接返回\"当前无待修复的 PR\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 每次只处理一个 PR 的一批建议。\n" +
				"- 修复后尽量运行本地测试。",
		},
		{
			Name:            "PR 合并检查",
			IntervalMinutes: 120,
			RequirementType: "github_pr_review",
			AgentCode:       "",
			SortOrder:       6,
			MDContent: "你是项目的自动化协作助手。当前任务是：检查 PR 是否达到可合并状态。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 gh pr list --repo owner/repo --state open 获取 open PRs。\n" +
				"2. 对每个 PR，检查最近 3 小时内是否已有你的 /lgtm 评论。如有，跳过。\n" +
				"3. 检查：CI 是否通过（gh pr checks）、是否有至少一条 /lgtm 或类似批准评论、是否有未解决的修改建议。\n" +
				"4. 若满足合并条件，在 PR 下评论 /lgtm。\n" +
				"5. 如果没有满足条件的 PR，直接返回\"当前无可合并 PR\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 不实际执行合并操作（仅评论 /lgtm，由人类或后续流程触发合并）。\n" +
				"- 每次最多检查 2 个 PR。",
		},
		{
			Name:            "PR 文档补充",
			IntervalMinutes: 480,
			RequirementType: "github_doc",
			AgentCode:       "",
			SortOrder:       7,
			MDContent: "你是项目的自动化协作助手。当前任务是：根据 PR 代码变更补充或更新文档。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 gh pr list --repo owner/repo --state open 获取近期 24 小时内有代码 push 的 PR。\n" +
				"2. 检查每个 PR 最近 6 小时内是否已有你的文档相关评论或 commit。如有，跳过（避免与修改修复冲突）。\n" +
				"3. 查看 PR diff，判断是否需要更新 README、API 文档、变更日志等。\n" +
				"4. 若需要，在 PR 分支上补充文档并 push，在 PR 下评论说明更新的内容。\n" +
				"5. 如果无需补充文档或全部处于冷却期，直接返回\"当前 PR 无需补充文档\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 每次最多处理 1 个 PR。\n" +
				"- 文档变更应与代码变更对应。",
		},
		{
			Name:            "PR 测试补充",
			IntervalMinutes: 480,
			RequirementType: "github_test",
			AgentCode:       "",
			SortOrder:       8,
			MDContent: "你是项目的自动化协作助手。当前任务是：根据 PR 代码变更补充测试用例。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 gh pr list --repo owner/repo --state open 获取 open PRs。\n" +
				"2. 检查每个 PR 最近 6 小时内是否已有你的测试相关评论或 commit。如有，跳过（避免与修改修复冲突）。\n" +
				"3. 查看 PR diff，识别新增/修改的功能点，判断是否需要补充单元测试、集成测试。\n" +
				"4. 若需要，在 PR 分支上编写并补充相关测试，运行测试确保通过，push 更新。\n" +
				"5. 在 PR 下评论说明补充的测试内容。\n" +
				"6. 如果所有 PR 测试都已充足或全部处于冷却期，直接返回\"当前 PR 无需补充测试\"。\n\n" +
				"## 约束\n" +
				"- 使用 gh CLI 操作 GitHub。\n" +
				"- 每次最多处理 1 个 PR。\n" +
				"- 补充的测试必须通过本地运行。",
		},
	}

	scenario, _ := domain.NewHeartbeatScenario(
		domain.NewHeartbeatScenarioID(id),
		"github_dev_workflow",
		"GitHub 开发协作工作流",
		"覆盖从 Issue 分析到 PR 合并的完整 GitHub 开发协作流水线",
		items,
	)
	scenario.SetIsBuiltIn(true)
	return scenario
}

// BuildAMCDevWorkflowScenario 构建 AMC 开发协作内置场景
func BuildAMCDevWorkflowScenario(id string) *domain.HeartbeatScenario {
	items := []domain.HeartbeatScenarioItem{
		{
			Name:            "Issue 分析",
			IntervalMinutes: 180,
			RequirementType: "amc_issue",
			AgentCode:       "",
			SortOrder:       1,
			MDContent: "你是项目的自动化协作助手。当前任务是：分析项目中的 open issues。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 amc issue list -R owner/repo 获取 open issues。\n" +
				"2. 对每个 issue，检查最近 3 小时内是否已有你的评论。如有，跳过该 issue（冷却机制）。\n" +
				"3. 对未被跳过的 issue，clone 仓库到临时目录，结合代码库分析 issue 描述。\n" +
				"4. 将分析结论（问题根因、可能影响的文件、建议修复方向）以评论形式发布到该 issue 下。\n" +
				"5. 如果没有 open issues，或全部处于冷却期，直接返回\"当前无待分析 issue\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 不修改任何源代码。\n" +
				"- 每次最多分析 2 个 issue。",
		},
		{
			Name:            "LGTM 代码编写",
			IntervalMinutes: 120,
			RequirementType: "amc_coding",
			AgentCode:       "",
			SortOrder:       2,
			MDContent: "你是项目的自动化协作助手。当前任务是：为已评审通过的 issue 编写代码并创建 PR。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 amc issue list -R owner/repo --label lgtm 获取已评审通过的 issue。\n" +
				"2. 检查最旧的一个 issue 最近 3 小时内是否已有你的评论或关联 PR。如有，跳过。\n" +
				"3. 对选中的 issue：clone 仓库、创建 feature 分支、根据 issue 描述实现代码修改。\n" +
				"4. 如有测试命令，运行基础测试确保通过。\n" +
				"5. push 分支后使用 amc pr create 创建 PR，并在描述中通过 Closes #issue_number 关联 issue。\n" +
				"6. 如果没有带 lgtm 标签的 open issue，或全部处于冷却期，直接返回\"当前无待编写代码的 issue\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 每次心跳只处理一个 issue。\n" +
				"- 如遇到编译/测试失败，在 PR 或 issue 下评论说明，不强制提交。",
		},
		{
			Name:            "PR 需求评审",
			IntervalMinutes: 120,
			RequirementType: "amc_pr_review",
			AgentCode:       "",
			SortOrder:       3,
			MDContent: "你是项目的自动化协作助手。当前任务是：检查 open PR 是否已完成需求评审。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 amc pr list -R owner/repo 获取 open PRs。\n" +
				"2. 对每个 PR，先检查最近 3 小时内是否已有你的评论。如有，跳过。\n" +
				"3. 再检查评论中是否包含\"需求评审通过\"字样。若已存在，跳过。\n" +
				"4. 若不存在，通过 PR 描述或 API 查找关联的 issue，阅读 issue 内容后在 PR 下评论：\n" +
				"   - 原始需求摘要\n" +
				"   - 需求评审结论（通过/需补充信息）\n" +
				"5. 如果所有 PR 都已通过需求评审或处于冷却期，直接返回\"所有 PR 已完成需求评审\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 不修改任何源代码。\n" +
				"- 每次最多评审 2 个 PR。",
		},
		{
			Name:            "PR 代码质量评审",
			IntervalMinutes: 180,
			RequirementType: "amc_pr_review",
			AgentCode:       "",
			SortOrder:       4,
			MDContent: "你是项目的自动化协作助手。当前任务是：对已通过需求评审的 open PR 进行代码质量和安全评审。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 amc pr list -R owner/repo 获取 open PRs。\n" +
				"2. 对每个 PR，检查是否已有\"需求评审通过\"评论。若无，跳过该 PR（阶段闸门）。\n" +
				"3. 检查该 PR 最近 3 小时内是否已有你的评论。如有，跳过（冷却机制）。\n" +
				"4. 使用 amc pr view 和 amc pr diff 查看变更，从代码质量、潜在 bug、安全漏洞、性能问题等角度进行评审。\n" +
				"5. 将评审意见以评论形式发布到 PR 下。\n" +
				"6. 如果没有符合条件的 PR，直接返回\"当前无待评审 PR\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 不修改任何源代码。\n" +
				"- 每次最多评审 2 个 PR。",
		},
		{
			Name:            "PR 修改修复",
			IntervalMinutes: 240,
			RequirementType: "amc_coding",
			AgentCode:       "",
			SortOrder:       5,
			MDContent: "你是项目的自动化协作助手。当前任务是：修复 PR 中的修改建议。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 查找当前项目下由你（AI 助手）创建或关联的 open PR。\n" +
				"2. 检查每个 PR 是否有未解决的评审建议。若无，跳过。\n" +
				"3. 检查最近 3 小时内你是否已对该 PR 评论或 push。如有，跳过（冷却机制，给人类留出确认时间）。\n" +
				"4. 阅读 PR 评论中的修改建议，判断哪些是可执行的合理建议。\n" +
				"5. checkout PR 分支、按建议修改代码、commit、push 更新。\n" +
				"6. 在 PR 下评论说明已修复的内容。\n" +
				"7. 如果没有待修复建议或全部处于冷却期，直接返回\"当前无待修复的 PR\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 每次只处理一个 PR 的一批建议。\n" +
				"- 修复后尽量运行本地测试。",
		},
		{
			Name:            "PR 合并检查",
			IntervalMinutes: 120,
			RequirementType: "amc_pr_review",
			AgentCode:       "",
			SortOrder:       6,
			MDContent: "你是项目的自动化协作助手。当前任务是：检查 PR 是否达到可合并状态。\n\n" +
				"项目仓库：${project.git_repo_url}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 amc pr list -R owner/repo 获取 open PRs。\n" +
				"2. 对每个 PR，检查最近 3 小时内是否已有你的 /lgtm 评论。如有，跳过。\n" +
				"3. 检查：CI 是否通过（amc pr merge-status）、是否有至少一条 /lgtm 或类似批准评论、是否有未解决的修改建议。\n" +
				"4. 若满足合并条件，在 PR 下评论 /lgtm。\n" +
				"5. 如果没有满足条件的 PR，直接返回\"当前无可合并 PR\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 不实际执行合并操作（仅评论 /lgtm，由人类或后续流程触发合并）。\n" +
				"- 每次最多检查 2 个 PR。",
		},
		{
			Name:            "PR 文档补充",
			IntervalMinutes: 480,
			RequirementType: "amc_doc",
			AgentCode:       "",
			SortOrder:       7,
			MDContent: "你是项目的自动化协作助手。当前任务是：根据 PR 代码变更补充或更新文档。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 amc pr list -R owner/repo 获取近期 24 小时内有代码 push 的 PR。\n" +
				"2. 检查每个 PR 最近 6 小时内是否已有你的文档相关评论或 commit。如有，跳过（避免与修改修复冲突）。\n" +
				"3. 查看 PR diff，判断是否需要更新 README、API 文档、变更日志等。\n" +
				"4. 若需要，在 PR 分支上补充文档并 push，在 PR 下评论说明更新的内容。\n" +
				"5. 如果无需补充文档或全部处于冷却期，直接返回\"当前 PR 无需补充文档\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 每次最多处理 1 个 PR。\n" +
				"- 文档变更应与代码变更对应。",
		},
		{
			Name:            "PR 测试补充",
			IntervalMinutes: 480,
			RequirementType: "amc_test",
			AgentCode:       "",
			SortOrder:       8,
			MDContent: "你是项目的自动化协作助手。当前任务是：根据 PR 代码变更补充测试用例。\n\n" +
				"项目仓库：${project.git_repo_url}\n" +
				"默认分支：${project.default_branch}\n\n" +
				"## 执行步骤\n" +
				"1. 使用 amc pr list -R owner/repo 获取 open PRs。\n" +
				"2. 检查每个 PR 最近 6 小时内是否已有你的测试相关评论或 commit。如有，跳过（避免与修改修复冲突）。\n" +
				"3. 查看 PR diff，识别新增/修改的功能点，判断是否需要补充单元测试、集成测试。\n" +
				"4. 若需要，在 PR 分支上编写并补充相关测试，运行测试确保通过，push 更新。\n" +
				"5. 在 PR 下评论说明补充的测试内容。\n" +
				"6. 如果所有 PR 测试都已充足或全部处于冷却期，直接返回\"当前 PR 无需补充测试\"。\n\n" +
				"## 约束\n" +
				"- 使用 amc CLI 操作 AtomGit。\n" +
				"- 每次最多处理 1 个 PR。\n" +
				"- 补充的测试必须通过本地运行。",
		},
	}

	scenario, _ := domain.NewHeartbeatScenario(
		domain.NewHeartbeatScenarioID(id),
		"amc_dev_workflow",
		"AMC 开发协作工作流",
		"覆盖从 Issue 分析到 PR 合并的完整 AMC 开发协作流水线",
		items,
	)
	scenario.SetIsBuiltIn(true)
	return scenario
}

// scenarioItemsEqual 比较两个心跳场景项列表是否相等
func scenarioItemsEqual(a, b []domain.HeartbeatScenarioItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name ||
			a[i].IntervalMinutes != b[i].IntervalMinutes ||
			a[i].RequirementType != b[i].RequirementType ||
			a[i].MDContent != b[i].MDContent {
			return false
		}
	}
	return true
}
