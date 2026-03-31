/**
 * CLI 工具 - 任务管理 CLI
 */
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/config"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	"go.uber.org/zap"
)

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin123"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-admin":
		createAdminUser(logger)
	case "delete-admin":
		deleteAdminUser(logger)
	case "requirement":
		handleRequirementCommand(logger)
	case "agent":
		handleAgentCommand(logger)
	case "project":
		handleProjectCommand(logger)
	case "config":
		handleConfigCommand(logger)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: taskmanager <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  create-admin          创建默认管理员用户 (admin/admin123)")
	fmt.Println("  delete-admin          删除默认管理员用户")
	fmt.Println("  agent list            列出所有 Agent")
	fmt.Println("  project list          列出所有项目")
	fmt.Println("  requirement create    创建新需求")
	fmt.Println("  requirement update     更新需求")
	fmt.Println("  requirement dispatch  派发需求")
	fmt.Println("  requirement complete  完成需求（创建 PR 后调用）")
	fmt.Println("  requirement list      列出需求")
	fmt.Println("  requirement get       获取需求详情")
	fmt.Println("  requirement review    分析PR并创建需求")
	fmt.Println("  config init           初始化配置文件")
	fmt.Println("  config show           显示当前配置")
	fmt.Println("")
	fmt.Println("Configuration:")
	fmt.Println("  配置文件路径: ~/.taskmanager/config.yaml")
	fmt.Println("  环境变量: TASKMANAGER_CONFIG (配置文件的路径)")
	fmt.Println("  环境变量: TASKMANAGER_DB_PATH (数据库路径)")
	fmt.Println("  环境变量: TASKMANAGER_API_BASE_URL (API 地址)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  taskmanager create-admin")
	fmt.Println("  taskmanager delete-admin")
	fmt.Println("  taskmanager agent list")
	fmt.Println("  taskmanager project list")
	fmt.Println("  taskmanager requirement create --project-id <id> --title <title> --description <desc>")
	fmt.Println("  taskmanager requirement dispatch <requirement_id>")
	fmt.Println("  taskmanager requirement complete --id <id> --pr-url <url>")
	fmt.Println("  taskmanager config init")
	fmt.Println("  taskmanager config show")
}

func handleRequirementCommand(logger *zap.Logger) {
	if len(os.Args) < 3 {
		printRequirementUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "create":
		createRequirement(logger)
	case "update":
		updateRequirement(logger)
	case "dispatch":
		dispatchRequirement(logger)
	case "complete":
		completeRequirement(logger)
	case "list":
		listRequirements(logger)
	case "get":
		getRequirement(logger)
	case "review":
		reviewPR(logger)
	default:
		fmt.Printf("Unknown requirement subcommand: %s\n", os.Args[2])
		printRequirementUsage()
		os.Exit(1)
	}
}

func printRequirementUsage() {
	fmt.Println("Usage: taskmanager requirement <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  create    创建新需求")
	fmt.Println("  update    更新需求")
	fmt.Println("  dispatch  派发需求（从项目配置获取agent/channel/sessionkey）")
	fmt.Println("  complete  完成需求")
	fmt.Println("  list      列出需求（默认过滤心跳需求）")
	fmt.Println("  get       获取需求详情")
	fmt.Println("  review    分析PR并创建需求")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  taskmanager requirement create --project-id <id> --title <title> --description <desc>")
	fmt.Println("  taskmanager requirement update --id <id> --title <new-title>")
	fmt.Println("  taskmanager requirement dispatch <requirement_id>")
	fmt.Println("  taskmanager requirement complete --id <id> --pr-url <url>")
	fmt.Println("  taskmanager requirement list --project-id <id>")
	fmt.Println("  taskmanager requirement list --include-heartbeat  # 包含心跳需求")
	fmt.Println("  taskmanager requirement get --id <id>")
}

func getDBPath() string {
	return config.GetDatabasePath()
}

func getUserRepos(logger *zap.Logger) (domain.UserRepository, domain.IDGenerator, func()) {
	dbPath := getDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database", zap.String("path", dbPath), zap.Error(err))
	}

	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.String("path", dbPath), zap.Error(err))
	}

	idGenerator := utils.NewNanoIDGenerator(21)
	userRepo := _persistence.NewSQLiteUserRepository(db)

	cleanup := func() {
		db.Close()
	}

	return userRepo, idGenerator, cleanup
}

func getRequirementRepos(logger *zap.Logger) (domain.RequirementRepository, domain.ProjectRepository, *application.RequirementApplicationService, *application.RequirementDispatchService, func()) {
	dbPath := getDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database", zap.String("path", dbPath), zap.Error(err))
	}

	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.String("path", dbPath), zap.Error(err))
	}

	idGenerator := utils.NewNanoIDGenerator(21)
	agentRepo := _persistence.NewSQLiteAgentRepository(db)
	projectRepo := _persistence.NewSQLiteProjectRepository(db)
	requirementRepo := _persistence.NewSQLiteRequirementRepository(db)

	appService := application.NewRequirementApplicationService(requirementRepo, projectRepo, idGenerator, nil, nil)
	dispatchService := application.NewRequirementDispatchService(
		requirementRepo,
		projectRepo,
		agentRepo,
		nil, // taskService
		nil, // sessionService
		idGenerator,
		nil, // replicaAgentManager
		nil, // hookExecutor
	)

	cleanup := func() {
		db.Close()
	}

	return requirementRepo, projectRepo, appService, dispatchService, cleanup
}

func createAdminUser(logger *zap.Logger) {
	userRepo, idGen, cleanup := getUserRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 检查是否已存在
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		logger.Fatal("检查用户失败", zap.Error(err))
	}
	if existingUser != nil {
		logger.Info("管理员用户已存在", zap.String("username", DefaultAdminUsername))
		return
	}

	// 使用 application 层创建用户（会正确处理密码哈希）
	userService := application.NewUserApplicationService(userRepo, idGen)
	user, err := userService.CreateUser(ctx, application.CreateUserCommand{
		Username:    DefaultAdminUsername,
		DisplayName: "系统管理员",
		Email:       "admin@local.dev",
		Password:    DefaultAdminPassword,
	})
	if err != nil {
		logger.Fatal("创建管理员用户失败", zap.Error(err))
	}

	logger.Info("管理员用户创建成功",
		zap.String("username", user.Username()),
		zap.String("userCode", user.UserCode().String()),
	)
	fmt.Printf("初始密码: %s (请登录后立即修改)\n", DefaultAdminPassword)
}

func deleteAdminUser(logger *zap.Logger) {
	userRepo, _, cleanup := getUserRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 查找用户
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		logger.Fatal("查找用户失败", zap.Error(err))
	}
	if existingUser == nil {
		logger.Info("管理员用户不存在", zap.String("username", DefaultAdminUsername))
		return
	}

	// 删除用户
	if err := userRepo.Delete(ctx, existingUser.ID()); err != nil {
		logger.Fatal("删除管理员用户失败", zap.Error(err))
	}

	logger.Info("管理员用户已删除", zap.String("username", DefaultAdminUsername))
}

// createRequirement 创建新需求
// 用法: taskmanager requirement create --project-id <id> --title <title> [--description <desc>] [--acceptance-criteria <criteria>] [--temp-workspace-root <path>]
func createRequirement(logger *zap.Logger) {
	projectIDPtr := flag.String("project-id", "", "项目 ID (必填)")
	titlePtr := flag.String("title", "", "需求标题 (必填)")
	descriptionPtr := flag.String("description", "", "需求描述")
	acceptanceCriteriaPtr := flag.String("acceptance-criteria", "", "验收标准")
	tempWorkspaceRootPtr := flag.String("temp-workspace-root", "", "临时工作目录根路径")

	flag.CommandLine.Parse(os.Args[3:])

	if *projectIDPtr == "" || *titlePtr == "" {
		fmt.Println("错误: --project-id 和 --title 是必填参数")
		fmt.Println("用法: taskmanager requirement create --project-id <id> --title <title> [--description <desc>]")
		os.Exit(1)
	}

	_, _, appService, _, cleanup := getRequirementRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 创建需求
	requirement, err := appService.CreateRequirement(ctx, application.CreateRequirementCommand{
		ProjectID:          domain.NewProjectID(*projectIDPtr),
		Title:              *titlePtr,
		Description:        *descriptionPtr,
		AcceptanceCriteria: *acceptanceCriteriaPtr,
		TempWorkspaceRoot:  *tempWorkspaceRootPtr,
	})
	if err != nil {
		logger.Fatal("创建需求失败", zap.Error(err))
	}

	logger.Info("需求创建成功",
		zap.String("requirement_id", requirement.ID().String()),
		zap.String("title", requirement.Title()),
	)
	fmt.Printf("需求创建成功！\nID: %s\n标题: %s\n", requirement.ID().String(), requirement.Title())
}

// updateRequirement 更新需求
// 用法: taskmanager requirement update --id <id> [--title <title>] [--description <desc>]
func updateRequirement(logger *zap.Logger) {
	idPtr := flag.String("id", "", "需求 ID (必填)")
	titlePtr := flag.String("title", "", "需求标题")
	descriptionPtr := flag.String("description", "", "需求描述")
	acceptanceCriteriaPtr := flag.String("acceptance-criteria", "", "验收标准")

	flag.CommandLine.Parse(os.Args[3:])

	if *idPtr == "" {
		fmt.Println("错误: --id 是必填参数")
		fmt.Println("用法: taskmanager requirement update --id <id> [--title <title>]")
		os.Exit(1)
	}

	requirementRepo, _, _, _, cleanup := getRequirementRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 查找需求
	requirementID := domain.NewRequirementID(*idPtr)
	requirement, err := requirementRepo.FindByID(ctx, requirementID)
	if err != nil {
		logger.Fatal("查找需求失败", zap.Error(err))
	}
	if requirement == nil {
		logger.Fatal("需求不存在", zap.String("id", *idPtr))
	}

	// 更新字段
	if *titlePtr != "" || *descriptionPtr != "" || *acceptanceCriteriaPtr != "" {
		newTitle := *titlePtr
		if newTitle == "" {
			newTitle = requirement.Title()
		}
		newDesc := *descriptionPtr
		if newDesc == "" {
			newDesc = requirement.Description()
		}
		newCriteria := *acceptanceCriteriaPtr
		if newCriteria == "" {
			newCriteria = requirement.AcceptanceCriteria()
		}
		if err := requirement.UpdateContent(newTitle, newDesc, newCriteria, requirement.TempWorkspaceRoot()); err != nil {
			logger.Fatal("更新需求失败", zap.Error(err))
		}
		if err := requirementRepo.Save(ctx, requirement); err != nil {
			logger.Fatal("保存需求失败", zap.Error(err))
		}
		logger.Info("需求更新成功",
			zap.String("requirement_id", requirement.ID().String()),
			zap.String("title", requirement.Title()),
		)
		fmt.Printf("需求更新成功！\nID: %s\n标题: %s\n", requirement.ID().String(), requirement.Title())
	}
}

// dispatchRequirement 派发需求
// 用法: taskmanager requirement dispatch <requirement_id>
// agent/channel/sessionkey 从项目配置中获取
func dispatchRequirement(logger *zap.Logger) {
	flag.CommandLine.Parse(os.Args[3:])

	args := flag.CommandLine.Args()
	if len(args) < 1 {
		fmt.Println("错误: 缺少必填参数")
		fmt.Println("")
		fmt.Println("用法: taskmanager requirement dispatch <requirement_id>")
		fmt.Println("")
		fmt.Println("示例: taskmanager requirement dispatch y6Wfok055CoE2twsr8dtD")
		os.Exit(1)
	}

	requirementID := args[0]

	// 1. 获取需求和项目信息（从数据库直接读取）
	requirementRepo, projectRepo, _, _, cleanup := getRequirementRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 查找需求
	req, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(requirementID))
	if err != nil {
		logger.Fatal("查找需求失败", zap.Error(err))
	}
	if req == nil {
		logger.Fatal("需求不存在", zap.String("id", requirementID))
	}

	// 查找项目获取派发配置
	project, err := projectRepo.FindByID(ctx, req.ProjectID())
	if err != nil {
		logger.Fatal("查找项目失败", zap.Error(err))
	}
	if project == nil {
		logger.Fatal("项目不存在", zap.String("project_id", req.ProjectID().String()))
	}

	// 检查项目是否配置了派发信息
	agentCode := project.AgentCode()
	channelCode := project.DispatchChannelCode()
	sessionKey := project.DispatchSessionKey()

	if agentCode == "" || sessionKey == "" {
		logger.Fatal("项目未配置派发信息",
			zap.String("project_id", project.ID().String()),
			zap.String("project_name", project.Name()),
			zap.String("agent_code", agentCode),
			zap.String("session_key", sessionKey))
	}

	// 如果 channelCode 为空，使用默认值
	if channelCode == "" {
		channelCode = "feishu"
	}

	// 2. 登录获取 token
	token, err := login()
	if err != nil {
		logger.Fatal("登录失败", zap.Error(err))
	}

	// 3. 调用派发 API
	reqBody := map[string]string{
		"requirement_id": requirementID,
		"agent_code":    agentCode,
		"channel_code":  channelCode,
		"session_key":   sessionKey,
	}
	reqJSON, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/requirements/dispatch", bytes.NewBuffer(reqJSON))
	if err != nil {
		logger.Fatal("创建请求失败", zap.Error(err))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Fatal("派发请求失败", zap.Error(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Fatal("派发需求失败", zap.String("status", resp.Status), zap.String("body", string(body)))
	}

	var result struct {
		RequirementID   string `json:"requirement_id"`
		TaskID         string `json:"task_id"`
		WorkspacePath  string `json:"workspace_path"`
		ReplicaAgentCode string `json:"replica_agent_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Fatal("解析响应失败", zap.Error(err))
	}

	logger.Info("需求派发成功",
		zap.String("requirement_id", result.RequirementID),
		zap.String("task_id", result.TaskID),
		zap.String("workspace_path", result.WorkspacePath),
		zap.String("replica_agent_code", result.ReplicaAgentCode),
	)
	fmt.Printf("需求派发成功！\n需求ID: %s\n任务ID: %s\n工作空间: %s\n分身AgentCode: %s\n",
		result.RequirementID, result.TaskID, result.WorkspacePath, result.ReplicaAgentCode)
}

// login 登录获取 token
func login() (string, error) {
	reqBody := map[string]string{
		"username": DefaultAdminUsername,
		"password": DefaultAdminPassword,
	}
	reqJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/auth/login", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed: %s", string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

// completeRequirement 完成需求（创建 PR 后调用）
// 用法: taskmanager requirement complete --id <requirement_id> --pr-url <pr_url> [--branch <branch_name>]
func completeRequirement(logger *zap.Logger) {
	idPtr := flag.String("id", "", "需求 ID (必填)")
	prURLPtr := flag.String("pr-url", "", "PR URL (必填)")
	branchPtr := flag.String("branch", "", "分支名 (可选)")

	flag.CommandLine.Parse(os.Args[3:])

	if *idPtr == "" || *prURLPtr == "" {
		fmt.Println("错误: --id 和 --pr-url 是必填参数")
		fmt.Println("用法: taskmanager requirement complete --id <requirement_id> --pr-url <pr_url>")
		os.Exit(1)
	}

	// 1. 登录获取 token
	token, err := login()
	if err != nil {
		logger.Fatal("登录失败", zap.Error(err))
	}

	// 2. 调用完成需求 API
	reqBody := map[string]string{
		"requirement_id": *idPtr,
		"pr_url":        *prURLPtr,
		"branch_name":   *branchPtr,
	}
	reqJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/requirements/pr", bytes.NewBuffer(reqJSON))
	if err != nil {
		logger.Fatal("创建请求失败", zap.Error(err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Fatal("完成需求请求失败", zap.Error(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Fatal("完成需求失败", zap.String("status", resp.Status), zap.String("body", string(body)))
	}

	var result struct {
		RequirementID string `json:"requirement_id"`
		PRURL        string `json:"pr_url"`
		BranchName   string `json:"branch_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Fatal("解析响应失败", zap.Error(err))
	}

	logger.Info("需求已完成",
		zap.String("requirement_id", result.RequirementID),
		zap.String("pr_url", result.PRURL),
		zap.String("branch", result.BranchName),
	)
	fmt.Printf("需求 %s 已标记为完成，PR: %s\n", result.RequirementID, result.PRURL)
}

// reviewPR 分析PR并创建需求
// 用法: taskmanager requirement review <pr_url> <project_id> [--title <title>]
// PR URL 格式: https://github.com/owner/repo/pull/123
// 或者只传 owner/repo 和 PR号: owner/repo 123
func reviewPR(logger *zap.Logger) {
	flag.CommandLine.Parse(os.Args[3:])
	args := flag.CommandLine.Args()

	if len(args) < 2 {
		fmt.Println("错误: 缺少必填参数")
		fmt.Println("")
		fmt.Println("用法: taskmanager requirement review <pr_url> <project_id> [--title <title>]")
		fmt.Println("   或: taskmanager requirement review <owner/repo> <pr_number> <project_id>")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  taskmanager requirement review https://github.com/owner/repo/pull/123 prj_xxx")
		fmt.Println("  taskmanager requirement review owner/repo 123 prj_xxx")
		fmt.Println("  taskmanager requirement review owner/repo 123 prj_xxx --title '修复登录bug'")
		os.Exit(1)
	}

	var prURL, owner, repo string
	var prNumber int

	// 解析参数
	if strings.HasPrefix(args[0], "http") {
		// 完整URL格式
		prURL = args[0]
		// 从URL中提取owner/repo
		parts := strings.Split(strings.TrimSuffix(args[0], "/"), "/")
		if len(parts) >= 2 {
			owner = parts[len(parts)-4]
			repo = parts[len(parts)-3]
		}
	} else {
		// owner/repo 格式
		ownerRepo := args[0]
		prNumberStr := args[1]

		parts := strings.Split(ownerRepo, "/")
		if len(parts) != 2 {
			fmt.Println("错误: owner/repo 格式不正确")
			os.Exit(1)
		}
		owner = parts[0]
		repo = parts[1]

		fmt.Sscanf(prNumberStr, "%d", &prNumber)
		prURL = fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, prNumber)
	}

	projectID := args[len(args)-1]

	titlePtr := flag.String("title", "", "需求标题 (可选)")

	// 1. 登录获取 token
	token, err := login()
	if err != nil {
		logger.Fatal("登录失败", zap.Error(err))
	}

	// 2. 获取PR信息
	prInfo, err := fetchPRInfo(owner, repo, prNumber)
	if err != nil {
		logger.Fatal("获取PR信息失败", zap.Error(err))
	}

	// 3. 获取PR评论
	comments, err := fetchPRComments(owner, repo, prNumber)
	if err != nil {
		logger.Warn("获取PR评论失败", zap.Error(err))
		comments = []PRComment{}
	}

	// 4. 生成需求内容
	title := *titlePtr
	if title == "" {
		title = fmt.Sprintf("PR #%d: %s", prNumber, prInfo.Title)
	}

	description := "## PR 信息\n\n"
	description += fmt.Sprintf("- PR: %s\n", prURL)
	description += fmt.Sprintf("- 标题: %s\n", prInfo.Title)
	description += fmt.Sprintf("- 作者: %s\n", prInfo.Author)
	description += fmt.Sprintf("- 状态: %s\n", prInfo.State)
	description += fmt.Sprintf("- 创建时间: %s\n", prInfo.CreatedAt)
	description += "\n## PR 描述\n\n" + prInfo.Body + "\n\n"

	if len(comments) > 0 {
		description += "## PR 评论\n\n"
		for _, c := range comments {
			description += fmt.Sprintf("### %s (%s)\n%s\n\n", c.Author, c.CreatedAt, c.Body)
		}
	}

	// 5. 调用创建需求 API
	reqBody := map[string]string{
		"project_id":          projectID,
		"title":              title,
		"description":        description,
		"acceptance_criteria": "根据PR评论内容确定验收标准",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/requirements", bytes.NewBuffer(reqJSON))
	if err != nil {
		logger.Fatal("创建请求失败", zap.Error(err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Fatal("创建需求请求失败", zap.Error(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		logger.Fatal("创建需求失败", zap.String("status", resp.Status), zap.String("body", string(body)))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Fatal("解析响应失败", zap.Error(err))
	}

	logger.Info("需求创建成功",
		zap.String("requirement_id", result.ID),
		zap.String("pr_url", prURL),
	)
	fmt.Printf("需求 %s 已创建 (来源: PR %s)\n", result.ID, prURL)
}

// PRInfo PR信息
type PRInfo struct {
	Title     string
	Body      string
	Author    string
	State     string
	CreatedAt string
}

// PRComment PR评论
type PRComment struct {
	Author    string
	Body      string
	CreatedAt string
}

// fetchPRInfo 获取PR信息
func fetchPRInfo(owner, repo string, prNumber int) (*PRInfo, error) {
	// 使用 gh 命令获取PR信息
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%s/%s#%d", owner, repo, prNumber),
		"--json", "title,body,author,state,createdAt", "-q", ".")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR info: %w", err)
	}

	var prInfo PRInfo
	if err := json.Unmarshal(output, &prInfo); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %w", err)
	}
	return &prInfo, nil
}

// fetchPRComments 获取PR评论
func fetchPRComments(owner, repo string, prNumber int) ([]PRComment, error) {
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, prNumber),
		"--jq", ".[] | {author: .user.login, body: .body, createdAt: .created_at}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR comments: %w", err)
	}

	var comments []PRComment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse PR comments: %w", err)
	}
	return comments, nil
}

// listRequirements 列出需求
// 用法: taskmanager requirement list [--project-id <id>] [--include-heartbeat]
func listRequirements(logger *zap.Logger) {
	projectIDPtr := flag.String("project-id", "", "项目 ID (可选)")
	includeHeartbeatPtr := flag.Bool("include-heartbeat", false, "包含心跳需求（默认不显示）")

	flag.CommandLine.Parse(os.Args[3:])

	requirementRepo, _, _, _, cleanup := getRequirementRepos(logger)
	defer cleanup()

	ctx := context.Background()

	var requirements []*domain.Requirement
	var err error

	if *projectIDPtr != "" {
		requirements, err = requirementRepo.FindByProjectID(ctx, domain.NewProjectID(*projectIDPtr))
	} else {
		// 查找所有项目
		allReqs, err := requirementRepo.FindAll(ctx)
		if err != nil {
			logger.Fatal("列出需求失败", zap.Error(err))
		}
		requirements = allReqs
	}

	if err != nil {
		logger.Fatal("列出需求失败", zap.Error(err))
	}

	// 过滤掉心跳需求（除非指定 --include-heartbeat）
	// 心跳需求有两种：1) requirement_type=heartbeat  2) 标题以[心跳]开头（旧数据）
	if !*includeHeartbeatPtr {
		filtered := make([]*domain.Requirement, 0)
		for _, req := range requirements {
			if req.RequirementType() == domain.RequirementTypeHeartbeat {
				continue
			}
			// 兼容旧数据：标题以[心跳]开头也算心跳需求
			if strings.HasPrefix(req.Title(), "[心跳]") {
				continue
			}
			filtered = append(filtered, req)
		}
		requirements = filtered
	}

	fmt.Printf("\n需求列表 (共 %d 个):\n", len(requirements))
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-20s %-10s %-10s %s\n", "ID", "状态", "开发状态", "标题")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, req := range requirements {
		fmt.Printf("%-20s %-10s %-10s %s\n",
			req.ID().String()[:16]+"...",
			req.Status(),
			req.DevState(),
			req.Title())
	}
	fmt.Println()
}

// getRequirement 获取需求详情
// 用法: taskmanager requirement get --id <id>
func getRequirement(logger *zap.Logger) {
	idPtr := flag.String("id", "", "需求 ID (必填)")

	flag.CommandLine.Parse(os.Args[3:])

	if *idPtr == "" {
		fmt.Println("错误: --id 是必填参数")
		fmt.Println("用法: taskmanager requirement get --id <id>")
		os.Exit(1)
	}

	requirementRepo, _, _, _, cleanup := getRequirementRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 查找需求
	requirementID := domain.NewRequirementID(*idPtr)
	requirement, err := requirementRepo.FindByID(ctx, requirementID)
	if err != nil {
		logger.Fatal("查找需求失败", zap.Error(err))
	}
	if requirement == nil {
		logger.Fatal("需求不存在", zap.String("id", *idPtr))
	}

	fmt.Println("\n需求详情:")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("ID: %s\n", requirement.ID().String())
	fmt.Printf("项目ID: %s\n", requirement.ProjectID().String())
	fmt.Printf("标题: %s\n", requirement.Title())
	fmt.Printf("描述: %s\n", requirement.Description())
	fmt.Printf("验收标准: %s\n", requirement.AcceptanceCriteria())
	fmt.Printf("状态: %s / %s\n", requirement.Status(), requirement.DevState())
	fmt.Printf("类型: %s\n", requirement.RequirementType())
	fmt.Printf("工作目录: %s\n", requirement.WorkspacePath())
	fmt.Printf("PR URL: %s\n", requirement.PRURL())
	fmt.Printf("分支: %s\n", requirement.BranchName())
	if requirement.StartedAt() != nil {
		fmt.Printf("开始时间: %s\n", requirement.StartedAt().Format("2006-01-02 15:04:05"))
	}
	if requirement.CompletedAt() != nil {
		fmt.Printf("完成时间: %s\n", requirement.CompletedAt().Format("2006-01-02 15:04:05"))
	}
	if reqResult := requirement.ClaudeRuntimeResult(); reqResult != "" {
		resultPreview := reqResult
		if len(resultPreview) > 100 {
			resultPreview = resultPreview[:100] + "..."
		}
		fmt.Printf("Claude执行结果: %s\n", resultPreview)
	}
	fmt.Println()
}

// handleAgentCommand 处理 agent 子命令
func handleAgentCommand(logger *zap.Logger) {
	if len(os.Args) < 3 {
		printAgentUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "list":
		listAgents(logger)
	default:
		fmt.Printf("Unknown agent subcommand: %s\n", os.Args[2])
		printAgentUsage()
		os.Exit(1)
	}
}

func printAgentUsage() {
	fmt.Println("Usage: taskmanager agent <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  list   列出所有 Agent")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  taskmanager agent list")
}

func getAgentRepos(logger *zap.Logger) (domain.AgentRepository, func()) {
	dbPath := getDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database", zap.String("path", dbPath), zap.Error(err))
	}

	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.String("path", dbPath), zap.Error(err))
	}

	agentRepo := _persistence.NewSQLiteAgentRepository(db)

	cleanup := func() {
		db.Close()
	}

	return agentRepo, cleanup
}

// listAgents 列出所有 Agent
func listAgents(logger *zap.Logger) {
	agentRepo, cleanup := getAgentRepos(logger)
	defer cleanup()

	ctx := context.Background()

	agents, err := agentRepo.FindAll(ctx)
	if err != nil {
		logger.Fatal("列出 Agent 失败", zap.Error(err))
	}

	fmt.Printf("\nAgent 列表 (共 %d 个):\n", len(agents))
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-20s %-15s %-15s %s\n", "ID", "类型", "状态", "名称")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, agent := range agents {
		agentType := "Unknown"
		if agent.AgentType() == domain.AgentTypeCoding {
			agentType = "CodingAgent"
		} else if agent.AgentType() == domain.AgentTypeBareLLM {
			agentType = "BareLLM"
		}

		status := "禁用"
		if agent.IsActive() {
			status = "启用"
		}

		name := agent.Name()
		if name == "" {
			name = "(无名称)"
		}

		fmt.Printf("%-20s %-15s %-15s %s\n",
			agent.ID().String()[:16]+"...",
			agentType,
			status,
			name)
	}
	fmt.Println()
}

// handleProjectCommand 处理 project 子命令
func handleProjectCommand(logger *zap.Logger) {
	if len(os.Args) < 3 {
		printProjectUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "list":
		listProjects(logger)
	default:
		fmt.Printf("Unknown project subcommand: %s\n", os.Args[2])
		printProjectUsage()
		os.Exit(1)
	}
}

func printProjectUsage() {
	fmt.Println("Usage: taskmanager project <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  list   列出所有项目")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  taskmanager project list")
}

func getProjectRepos(logger *zap.Logger) (domain.ProjectRepository, func()) {
	dbPath := getDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database", zap.String("path", dbPath), zap.Error(err))
	}

	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.String("path", dbPath), zap.Error(err))
	}

	projectRepo := _persistence.NewSQLiteProjectRepository(db)

	cleanup := func() {
		db.Close()
	}

	return projectRepo, cleanup
}

// listProjects 列出所有项目
func listProjects(logger *zap.Logger) {
	projectRepo, cleanup := getProjectRepos(logger)
	defer cleanup()

	ctx := context.Background()

	projects, err := projectRepo.FindAll(ctx)
	if err != nil {
		logger.Fatal("列出项目失败", zap.Error(err))
	}

	fmt.Printf("\n项目列表 (共 %d 个):\n", len(projects))
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-20s %s\n", "ID", "名称")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, project := range projects {
		fmt.Printf("%-20s %s\n",
			project.ID().String()[:16]+"...",
			project.Name())
	}
	fmt.Println()
}

// handleConfigCommand 处理 config 子命令
func handleConfigCommand(logger *zap.Logger) {
	if len(os.Args) < 3 {
		printConfigUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "init":
		initConfig(logger)
	case "show":
		showConfig(logger)
	default:
		fmt.Printf("Unknown config subcommand: %s\n", os.Args[2])
		printConfigUsage()
		os.Exit(1)
	}
}

func printConfigUsage() {
	fmt.Println("Usage: taskmanager config <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  init   初始化配置文件 (~/.taskmanager/config.yaml)")
	fmt.Println("  show   显示当前配置")
	fmt.Println("")
	fmt.Println("Configuration file location (priority):")
	fmt.Println("  1. $TASKMANAGER_CONFIG environment variable")
	fmt.Println("  2. ./taskmanager.yaml (current directory)")
	fmt.Println("  3. ~/.taskmanager/config.yaml (home directory)")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  TASKMANAGER_CONFIG      - Config file path")
	fmt.Println("  TASKMANAGER_DB_PATH     - Database file path")
	fmt.Println("  TASKMANAGER_API_BASE_URL - API base URL")
}

// initConfig 初始化配置文件
func initConfig(logger *zap.Logger) {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Fatal("Failed to get home directory", zap.Error(err))
	}

	configPath := filepath.Join(home, ".taskmanager", "config.yaml")

	if err := config.WriteDefaultConfig(configPath); err != nil {
		logger.Fatal("Failed to write config file", zap.String("path", configPath), zap.Error(err))
	}

	fmt.Printf("配置文件已创建: %s\n", configPath)
	fmt.Println("")
	fmt.Println("请编辑配置文件设置数据库路径:")
	fmt.Printf("  vim %s\n", configPath)
}

// showConfig 显示当前配置
func showConfig(logger *zap.Logger) {
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	fmt.Println("当前配置:")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("Database Path: %s\n", cfg.Database.Path)
	fmt.Printf("API Base URL: %s\n", cfg.API.BaseURL)
	fmt.Printf("Log Level: %s\n", cfg.Logging.Level)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("")
	fmt.Println("配置加载来源:")
	configPath := ""
	if p := os.Getenv("TASKMANAGER_CONFIG"); p != "" {
		configPath = fmt.Sprintf("TASKMANAGER_CONFIG=%s", p)
	} else {
		home, _ := os.UserHomeDir()
		defaultPath := filepath.Join(home, ".taskmanager", "config.yaml")
		if _, err := os.Stat(defaultPath); err == nil {
			configPath = fmt.Sprintf("~/.taskmanager/config.yaml (default)")
		} else {
			configPath = "无配置文件，使用环境变量或默认值"
		}
	}
	fmt.Printf("  %s\n", configPath)
	fmt.Println("")
	fmt.Println("环境变量覆盖:")
	if dbPath := os.Getenv("TASKMANAGER_DB_PATH"); dbPath != "" {
		fmt.Printf("  TASKMANAGER_DB_PATH=%s\n", dbPath)
	}
	if apiURL := os.Getenv("TASKMANAGER_API_BASE_URL"); apiURL != "" {
		fmt.Printf("  TASKMANAGER_API_BASE_URL=%s\n", apiURL)
	}
}
