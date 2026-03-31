package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var requirementTasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "查看需求的子任务",
	Example: `  taskmanager requirement tasks --id <requirement_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Println("错误: --id 是必填参数")
			cmd.Usage()
			return
		}

		requirementRepo, _, _, _, reqCleanup := getRequirementRepos()
		defer reqCleanup()

		taskRepo, taskCleanup := getTaskRepo()
		defer taskCleanup()

		ctx := context.Background()

		// 查找需求
		requirement, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(id))
		if err != nil {
			fmt.Printf("查找需求失败: %v\n", err)
			return
		}
		if requirement == nil {
			fmt.Printf("需求不存在: %s\n", id)
			return
		}

		// 显示需求基本信息
		fmt.Println("\n需求信息:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("ID: %s\n", requirement.ID().String())
		fmt.Printf("标题: %s\n", requirement.Title())
		fmt.Printf("状态: %s / %s\n", requirement.Status(), requirement.DevState())
		fmt.Printf("SessionKey: %s\n", requirement.DispatchSessionKey())
		fmt.Printf("工作目录: %s\n", requirement.WorkspacePath())
		fmt.Printf("分身Agent: %s\n", requirement.ReplicaAgentCode())
		fmt.Println()

		// 尝试通过 session_key 查找相关任务
		sessionKey := requirement.DispatchSessionKey()
		if sessionKey == "" {
			fmt.Println("提示: 该需求尚未派发，没有关联的 session_key")
			fmt.Println()
			return
		}

		// 查找所有任务，然后筛选
		allTasks, err := taskRepo.FindAll(ctx)
		if err != nil {
			fmt.Printf("查找任务失败: %v\n", err)
			return
		}

		// 筛选与该需求相关的任务（通过 session_key 匹配）
		var relatedTasks []*domain.Task
		for _, task := range allTasks {
			if task.SessionKey() == sessionKey {
				relatedTasks = append(relatedTasks, task)
			}
		}

		if len(relatedTasks) == 0 {
			fmt.Println("子任务: 无")
			fmt.Println()
			return
		}

		// 显示子任务列表
		fmt.Printf("子任务 (共 %d 个):\n", len(relatedTasks))
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %-10s %-10s %s\n", "ID", "状态", "进度", "任务名称")
		fmt.Println("--------------------------------------------------------------------------------")

		for _, task := range relatedTasks {
			taskID := task.ID().String()
			if len(taskID) > 16 {
				taskID = taskID[:16] + "..."
			}
			fmt.Printf("%-20s %-10s %-10d %s\n",
				taskID,
				task.Status(),
				task.Progress().Value(),
				task.Name())
		}
		fmt.Println()

		// 显示顶层任务（有 parent_id 的为子任务，没有的为顶层任务）
		var topLevelTasks []*domain.Task
		for _, task := range relatedTasks {
			if task.ParentID() == nil {
				topLevelTasks = append(topLevelTasks, task)
			}
		}

		if len(topLevelTasks) > 0 {
			fmt.Printf("顶层任务 (共 %d 个):\n", len(topLevelTasks))
			fmt.Println("--------------------------------------------------------------------------------")

			for _, task := range topLevelTasks {
				// 查找该顶层任务的子任务
				var childTasks []*domain.Task
				for _, t := range relatedTasks {
					if t.ParentID() != nil && t.ParentID().String() == task.ID().String() {
						childTasks = append(childTasks, t)
					}
				}

				fmt.Printf("\n任务: %s\n", task.Name())
				fmt.Printf("  ID: %s\n", task.ID().String())
				fmt.Printf("  状态: %s | 进度: %d%%\n", task.Status(), task.Progress().Value())
				fmt.Printf("  类型: %s\n", task.Type())

				if len(childTasks) > 0 {
					fmt.Printf("  子任务 (%d个):\n", len(childTasks))
					for _, child := range childTasks {
						fmt.Printf("    - %s [%s, %d%%]\n", child.Name(), child.Status(), child.Progress().Value())
					}
				}
			}
			fmt.Println()
		}
	},
}

func init() {
	requirementTasksCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}
