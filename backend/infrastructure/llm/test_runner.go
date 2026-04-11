package llm

import (
	"context"
	"fmt"
	"time"
)

// TestLLMConnection tests an LLM connection by creating a client and sending a test message.
func TestLLMConnection(ctx context.Context, providerKey, model, apiKey, baseURL string) error {
	config := &Config{
		ProviderType: providerKey,
		Model:        model,
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Temperature:  0.7,
		MaxTokens:    1024,
	}

	client, err := NewLLMProvider(config)
	if err != nil {
		return fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err = client.Generate(testCtx, "Hi, please respond with 'OK' if you receive this message.")
	return err
}
