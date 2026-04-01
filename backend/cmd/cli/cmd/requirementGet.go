package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var requirementGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取需求详情",
	Example: `  taskmanager requirement get --id <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Println("错误: --id 是必填参数")
			cmd.Usage()
			return
		}

		requirementRepo, _, _, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx := context.Background()

		requirement, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(id))
		if err != nil {
			fmt.Printf("查找需求失败: %v\n", err)
			return
		}
		if requirement == nil {
			fmt.Printf("需求不存在: %s\n", id)
			return
		}

		printRequirementDetail(requirement)
	},
}

func printRequirementDetail(r *domain.Requirement) {
	fmt.Println("\n===================================== 需求详情 =====================================")
	fmt.Printf("ID:              %s\n", r.ID().String())
	fmt.Printf("项目ID:          %s\n", r.ProjectID().String())
	fmt.Println("-----------------------------------------------------------------------------------------------")
	fmt.Printf("标题:            %s\n", r.Title())
	fmt.Printf("描述:            %s\n", r.Description())
	fmt.Printf("验收标准:        %s\n", r.AcceptanceCriteria())
	fmt.Println("-----------------------------------------------------------------------------------------------")
	fmt.Printf("状态:            %s / %s\n", r.Status(), r.DevState())
	fmt.Printf("类型:            %s\n", r.RequirementType())
	fmt.Println("===================================== 派发信息 =====================================")
	fmt.Printf("工作目录:        %s\n", r.WorkspacePath())
	fmt.Printf("分身Agent:       %s\n", r.ReplicaAgentCode())
	fmt.Printf("派发SessionKey:  %s\n", r.DispatchSessionKey())
	fmt.Printf("分支:            %s\n", r.BranchName())
	fmt.Printf("PR URL:          %s\n", r.PRURL())
	fmt.Println("===================================== 时间信息 =====================================")
	fmt.Printf("创建时间:        %s\n", r.CreatedAt().Format("2006-01-02 15:04:05"))
	fmt.Printf("更新时间:        %s\n", r.UpdatedAt().Format("2006-01-02 15:04:05"))
	if r.StartedAt() != nil {
		fmt.Printf("开始时间:        %s\n", r.StartedAt().Format("2006-01-02 15:04:05"))
	}
	if r.CompletedAt() != nil {
		fmt.Printf("完成时间:        %s\n", r.CompletedAt().Format("2006-01-02 15:04:05"))
	}
	fmt.Println("===================================== Claude 执行状态 =====================================")
	if r.ClaudeRuntimePrompt() != "" {
		fmt.Println("-----------------------------------------------------------------------------------------------")
		fmt.Println("Claude执行提示词:")
		fmt.Println("-----------------------------------------------------------------------------------------------")
		fmt.Println(r.ClaudeRuntimePrompt())
	}
	fmt.Printf("Runtime状态:     %s\n", r.ClaudeRuntimeStatus())
	if r.ClaudeRuntimeStartedAt() != nil {
		fmt.Printf("Runtime开始:     %s\n", r.ClaudeRuntimeStartedAt().Format("2006-01-02 15:04:05"))
	}
	if r.ClaudeRuntimeEndedAt() != nil {
		fmt.Printf("Runtime结束:     %s\n", r.ClaudeRuntimeEndedAt().Format("2006-01-02 15:04:05"))
	}
	if r.ClaudeRuntimeError() != "" {
		fmt.Printf("Runtime错误:     %s\n", r.ClaudeRuntimeError())
	}
	if r.ClaudeRuntimeResult() != "" {
		fmt.Println("-----------------------------------------------------------------------------------------------")
		fmt.Println("Claude执行结果:")
		fmt.Println("-----------------------------------------------------------------------------------------------")
		fmt.Println(r.ClaudeRuntimeResult())
	}
	if r.LastError() != "" {
		fmt.Printf("最近错误:        %s\n", r.LastError())
	}
	fmt.Println("===============================================================================================")
}

func init() {
	requirementGetCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}