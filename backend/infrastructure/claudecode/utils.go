package claudecode

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

// getUsageInt extracts an int value from a usage map
func getUsageInt(usage map[string]any, key string) int {
	if v, ok := usage[key]; ok {
		if f, ok := toFloat64(v); ok {
			return int(f)
		}
	}
	return 0
}

// resolveProvider 解析 Agent 使用的 LLM Provider
