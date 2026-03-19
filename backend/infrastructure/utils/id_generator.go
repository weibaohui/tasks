/**
 * NanoID 生成器
 * 使用 nanoid 库生成唯一ID
 */
package utils

import (
	"github.com/aidarkhanov/nanoid/v2"
)

// NanoIDGenerator NanoID 生成器
type NanoIDGenerator struct {
	size int
}

// NewNanoIDGenerator 创建 NanoID 生成器
func NewNanoIDGenerator(size int) *NanoIDGenerator {
	if size <= 0 {
		size = 21
	}
	return &NanoIDGenerator{size: size}
}

// Generate 生成 ID
func (g *NanoIDGenerator) Generate() string {
	id, _ := nanoid.New()
	return id
}
