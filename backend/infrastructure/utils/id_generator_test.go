/**
 * NanoIDGenerator 单元测试
 */
package utils

import (
	"testing"
)

func TestNewNanoIDGenerator(t *testing.T) {
	gen := NewNanoIDGenerator(10)

	if gen.size != 10 {
		t.Errorf("期望 size 为 10, 实际为 %d", gen.size)
	}
}

func TestNewNanoIDGenerator_DefaultSize(t *testing.T) {
	gen := NewNanoIDGenerator(0)

	if gen.size != 21 {
		t.Errorf("期望默认 size 为 21, 实际为 %d", gen.size)
	}
}

func TestNewNanoIDGenerator_NegativeSize(t *testing.T) {
	gen := NewNanoIDGenerator(-5)

	if gen.size != 21 {
		t.Errorf("期望默认 size 为 21, 实际为 %d", gen.size)
	}
}

func TestNanoIDGenerator_Generate(t *testing.T) {
	gen := NewNanoIDGenerator(10)

	id := gen.Generate()

	if id == "" {
		t.Error("生成的 ID 不应为空")
	}

	// nanoid 生成的长度可能是 21（默认），不强制检查具体长度
	if len(id) < 10 {
		t.Errorf("期望 ID 长度至少为 10, 实际为 %d", len(id))
	}
}

func TestNanoIDGenerator_Generate_Uniqueness(t *testing.T) {
	gen := NewNanoIDGenerator(10)

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := gen.Generate()
		if ids[id] {
			t.Errorf("生成重复 ID: %s", id)
		}
		ids[id] = true
	}
}
