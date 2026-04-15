package domain

import (
	"encoding/json"
	"time"
)

// TodoItem 单个 todo 项
type TodoItem struct {
	Content   string `json:"content"`
	Status    string `json:"status"`
	Priority  string `json:"priority,omitempty"`
}

// ProgressData 进度数据（存储完整的 todo 列表和计算出的百分比）
type ProgressData struct {
	Items      []TodoItem `json:"items"`
	Percent    int        `json:"percent"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// NewProgressData 创建空的进度数据
func NewProgressData() *ProgressData {
	return &ProgressData{
		Items:     []TodoItem{},
		Percent:   0,
		UpdatedAt: time.Now(),
	}
}

// CalculatePercent 根据 items 状态重新计算百分比
func (p *ProgressData) CalculatePercent() {
	if len(p.Items) == 0 {
		p.Percent = 0
		return
	}
	completed := 0
	for _, item := range p.Items {
		if item.Status == "completed" || item.Status == "done" {
			completed++
		}
	}
	p.Percent = completed * 100 / len(p.Items)
	p.UpdatedAt = time.Now()
}

// ToJSON 序列化为 JSON 字符串
func (p *ProgressData) ToJSON() string {
	if p == nil {
		return ""
	}
	b, _ := json.Marshal(p)
	return string(b)
}

// ProgressDataFromJSON 从 JSON 字符串解析
func ProgressDataFromJSON(s string) *ProgressData {
	if s == "" {
		return NewProgressData()
	}
	var p ProgressData
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return NewProgressData()
	}
	return &p
}
