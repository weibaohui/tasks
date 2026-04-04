/**
 * 实体基础接口定义
 */
package domain

// Entity 实体接口，所有实体实现此接口
type Entity interface {
	// GetID 返回实体的唯一标识
	GetID() string
}
