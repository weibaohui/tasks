/**
 * LLM Provider 集成测试
 * 需要先 source .env 文件配置 API Key
 * 执行方式: cd backend && source .env && go test -v ./infrastructure/llm/... -run TestLLM
 */
package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLLM_Provider_Integration LLM Provider 集成测试
// 需要配置环境变量: OPENAI_API_KEY, OPENAI_BASE_URL
func TestLLM_Provider_Integration(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过测试: 未配置 OPENAI_API_KEY")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "Qwen/Qwen3-8B"
	}

	t.Logf("配置信息: baseURL=%s, model=%s", baseURL, model)

	config := &Config{
		ProviderType: "openai",
		Model:        model,
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Temperature:  0.7,
		MaxTokens:    1000,
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("创建 OpenAI Provider 失败: %v", err)
	}

	ctx := context.Background()

	// 测试 1: 简单的 Generate 调用
	t.Log("测试 1: 简单的 Generate 调用...")
	testGenerate(ctx, provider, t)

	// 间隔 3 秒
	t.Log("等待 3 秒...")
	time.Sleep(3 * time.Second)

	// 测试 2: 生成子任务计划
	t.Log("测试 2: 生成子任务计划...")
	testGenerateSubTasks(ctx, provider, t)
}

func testGenerate(ctx context.Context, provider *OpenAIProvider, t *testing.T) {
	prompt := "请用一句话介绍自己，只返回内容，不要其他解释。"

	result, err := provider.Generate(ctx, prompt)
	if err != nil {
		t.Fatalf("Generate 调用失败: %v", err)
	}

	if result == "" {
		t.Error("Generate 返回空结果")
	}

	t.Logf("Generate 结果: %s", result)
}

func testGenerateSubTasks(ctx context.Context, provider *OpenAIProvider, t *testing.T) {
	plan, err := provider.GenerateSubTasks(
		ctx,
		"数据处理任务",
		"处理用户销售数据并生成报表",
		1,
		4,
	)
	if err != nil {
		t.Fatalf("GenerateSubTasks 调用失败: %v", err)
	}

	if plan == nil {
		t.Fatal("GenerateSubTasks 返回 nil plan")
	}

	t.Logf("LLM 生成了 %d 个子任务", len(plan.SubTasks))
	if plan.Reason != "" {
		t.Logf("原因: %s", plan.Reason)
	}

	for i, st := range plan.SubTasks {
		t.Logf("  子任务 %d: goal=%s, type=%s", i+1, st.Goal, st.TaskType)
	}

	if len(plan.SubTasks) == 0 {
		t.Error("LLM 未生成任何子任务")
	}
}

// TestAgentHandler_Integration Agent Handler 集成测试
// 需要先 source .env 文件
func TestAgentHandler_Integration(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过测试: 未配置 OPENAI_API_KEY")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "Qwen/Qwen3-8B"
	}

	t.Logf("配置信息: baseURL=%s, model=%s", baseURL, model)

	config := &Config{
		ProviderType: "openai",
		Model:        model,
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Temperature:  0.7,
		MaxTokens:    1000,
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("创建 OpenAI Provider 失败: %v", err)
	}

	// 验证 provider 可以生成子任务
	t.Log("测试 Agent Handler 子任务生成能力...")

	ctx := context.Background()

	plan, err := provider.GenerateSubTasks(
		ctx,
		"用户数据分析",
		"分析用户行为数据，找出增长机会",
		1,
		4,
	)
	if err != nil {
		t.Fatalf("GenerateSubTasks 失败: %v", err)
	}

	t.Logf("Agent Handler 测试通过: 生成了 %d 个子任务", len(plan.SubTasks))
	for i, st := range plan.SubTasks {
		t.Logf("  子任务 %d: [%s] %s", i+1, st.TaskType, st.Goal)
	}
}
