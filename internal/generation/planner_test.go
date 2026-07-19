package generation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"manga-drama-studio/internal/newapi"
)

type plannerFixture struct{}

func (plannerFixture) ChatJSON(_ context.Context, model string, messages []newapi.Message, output any) error {
	if model != "deepseek-v4-pro" {
		return assert.AnError
	}
	if len(messages) == 0 {
		return assert.AnError
	}
	switch target := output.(type) {
	case *Script:
		*target = Script{Title: "雨夜", Logline: "画师救画", Scenes: []Scene{{ID: "scene-1", Summary: "相遇"}}}
	case *CharacterSheet:
		*target = CharacterSheet{Characters: []Character{{Name: "阿绘", Appearance: "黑发红衣"}}}
	case *Storyboard:
		*target = Storyboard{Shots: []Shot{{ID: "shot-001", Order: 1, DurationSeconds: 5, Ratio: "16:9", Prompt: "黑发红衣的阿绘说你好", Subtitle: "你好"}}}
	}
	return nil
}

func TestPlannerProducesValidatedPipelineDocuments(t *testing.T) {
	planner := NewPlanner(plannerFixture{}, "deepseek-v4-pro")
	script, err := planner.Script(context.Background(), "雨夜画师")
	require.NoError(t, err)
	characters, err := planner.Characters(context.Background(), script)
	require.NoError(t, err)
	storyboard, err := planner.Storyboard(context.Background(), script, characters, map[string]any{"ratio": "16:9"})
	require.NoError(t, err)
	assert.Equal(t, "shot-001", storyboard.Shots[0].ID)
}
