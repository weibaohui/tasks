package domain

// SkillInfo 技能信息
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Requires    string `json:"requires,omitempty"`
}

// SkillsLoader 技能加载器接口
type SkillsLoader interface {
	ListSkills() []SkillInfo
	LoadSkillContent(name string) string
	GetSkillMetadata(name string) map[string]string
	CheckRequirements(name string) bool
	GetMissingRequirements(name string) string
}
