/**
 * 领域工厂接口定义
 */
package domain

// LLMProviderFactory 基础设施层实现，用于创建实际的 LLM Provider
// Build 返回的 interface{} 是 infrastructure/llm.LLMProvider，由调用方进行类型断言
type LLMProviderFactory interface {
	// Build 根据配置创建 LLM Provider
	Build(config *LLMProviderConfig) (interface{}, error)
}
