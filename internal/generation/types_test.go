package generation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoryboardValidationProtectsVideoGenerationBounds(t *testing.T) {
	valid := Storyboard{Shots: []Shot{{ID: "shot-001", Order: 1, DurationSeconds: 5, Ratio: "16:9", Prompt: "电影镜头，角色说你好", Subtitle: "你好"}}}
	assert.NoError(t, valid.Validate())

	tests := []struct {
		name string
		edit func(*Storyboard)
	}{
		{"too many shots", func(s *Storyboard) {
			for len(s.Shots) <= 12 {
				s.Shots = append(s.Shots, s.Shots[0])
			}
		}},
		{"duration", func(s *Storyboard) { s.Shots[0].DurationSeconds = 0 }},
		{"ratio", func(s *Storyboard) { s.Shots[0].Ratio = "5:4" }},
		{"prompt", func(s *Storyboard) { s.Shots[0].Prompt = "" }},
		{"order", func(s *Storyboard) { s.Shots[0].Order = 2 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := valid
			value.Shots = append([]Shot(nil), valid.Shots...)
			tt.edit(&value)
			assert.Error(t, value.Validate())
		})
	}
}

func TestScriptAndCharacterValidation(t *testing.T) {
	assert.Error(t, (Script{}).Validate())
	assert.NoError(t, (Script{Title: "测试", Logline: "一句话", Scenes: []Scene{{ID: "scene-1", Summary: "场景"}}}).Validate())
	assert.Error(t, (CharacterSheet{}).Validate())
	assert.NoError(t, (CharacterSheet{Characters: []Character{{Name: "阿绘", Appearance: "黑发红衣"}}}).Validate())
}
