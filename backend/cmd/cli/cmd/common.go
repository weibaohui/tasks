package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/config"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	"go.uber.org/zap"
)

// printJSONError 安全地输出 JSON 格式的错误信息
func printJSONError(format string, args ...interface{}) {
	errResult := map[string]string{
		"error": fmt.Sprintf(format, args...),
	}
	jsonBytes, _ := json.Marshal(errResult)
	fmt.Print(string(jsonBytes))
}

var logger *zap.Logger
var (
	sharedDB   *sql.DB
	sharedDBMu sync.Mutex
)

func init() {
	logger, _ = zap.NewDevelopment()
}

func getDB() (*sql.DB, func()) {
	sharedDBMu.Lock()
	defer sharedDBMu.Unlock()

	if sharedDB != nil {
		return sharedDB, func() {}
	}

	dbPath := config.GetDatabasePath()
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}

	if err := _persistence.InitSchema(db); err != nil {
		fmt.Printf("Failed to init schema: %v\n", err)
		os.Exit(1)
	}

	sharedDB = db
	return db, func() {}
}

func getUserRepos() (domain.UserRepository, domain.IDGenerator, func()) {
	db, cleanup := getDB()
	idGenerator := utils.NewNanoIDGenerator(21)
	userRepo := _persistence.NewSQLiteUserRepository(db)
	return userRepo, idGenerator, func() {
		cleanup()
	}
}

func getRequirementRepos() (domain.RequirementRepository, domain.ProjectRepository, *application.RequirementApplicationService, *application.RequirementDispatchService, func()) {
	db, cleanup := getDB()
	idGenerator := utils.NewNanoIDGenerator(21)
	agentRepo := _persistence.NewSQLiteAgentRepository(db)
	projectRepo := _persistence.NewSQLiteProjectRepository(db)
	requirementRepo := _persistence.NewSQLiteRequirementRepository(db)

	appService := application.NewRequirementApplicationService(requirementRepo, projectRepo, idGenerator, nil, nil)
	dispatchService := application.NewRequirementDispatchService(
		requirementRepo,
		projectRepo,
		agentRepo,
		nil,
		nil,
		idGenerator,
		nil,
		nil,
	)

	return requirementRepo, projectRepo, appService, dispatchService, cleanup
}

func getProjectRepo() (domain.ProjectRepository, func()) {
	db, cleanup := getDB()
	projectRepo := _persistence.NewSQLiteProjectRepository(db)
	return projectRepo, cleanup
}

func getAgentRepo() (domain.AgentRepository, func()) {
	db, cleanup := getDB()
	agentRepo := _persistence.NewSQLiteAgentRepository(db)
	return agentRepo, cleanup
}

func getTaskRepo() (domain.TaskRepository, func()) {
	db, cleanup := getDB()
	taskRepo := _persistence.NewSQLiteTaskRepository(db)
	return taskRepo, cleanup
}

// AddCommands 注册所有子命令
func AddCommands() {
	rootCmd.AddCommand(createAdminCmd)
	rootCmd.AddCommand(deleteAdminCmd)
	rootCmd.AddCommand(requirementCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(configCmd)
}

// RegisterFlagErrorFunc 自定义错误处理
func RegisterFlagErrorFunc() {
	cobra.EnableCommandSorting = false
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return err
	})
}
