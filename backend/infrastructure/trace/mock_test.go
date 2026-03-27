/**
 * Mock ID Generator for Testing
 */
package trace

import (
	"sync/atomic"
)

// MockIDGenerator 生成确定性的 ID 用于测试
type MockIDGenerator struct {
	counter uint64
}

func NewMockIDGenerator() *MockIDGenerator {
	return &MockIDGenerator{counter: 0}
}

func (g *MockIDGenerator) Generate() string {
	v := atomic.AddUint64(&g.counter, 1)
	return "id-" + string(rune('a'+v-1)) + string(rune('0'+v-1))
}

func (g *MockIDGenerator) NewTraceID() string {
	v := atomic.AddUint64(&g.counter, 1)
	return "trace-" + u64toa(v)
}

func (g *MockIDGenerator) NewSpanID() string {
	v := atomic.AddUint64(&g.counter, 1)
	return "span-" + u64toa(v)
}

func u64toa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func (g *MockIDGenerator) Reset() {
	atomic.StoreUint64(&g.counter, 0)
}
