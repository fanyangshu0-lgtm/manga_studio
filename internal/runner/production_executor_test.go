package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"manga-drama-studio/internal/domain"
	"manga-drama-studio/internal/generation"
	"manga-drama-studio/internal/newapi"
)

type executorChatFixture struct{}

func (executorChatFixture) ChatJSON(_ context.Context, _ string, _ []newapi.Message, output any) error {
	*output.(*generation.Script) = generation.Script{Title: "标题", Logline: "梗概", Scenes: []generation.Scene{{ID: "s1", Summary: "场景"}}}
	return nil
}

func TestProductionExecutorUsesPlannerForScriptNode(t *testing.T) {
	executor := NewProductionExecutor(generation.NewPlanner(executorChatFixture{}, "deepseek-v4-pro"))
	result, err := executor.Execute(context.Background(), domain.Node{Type: "script"}, map[string]any{"input.story": "一个故事"}, func(int, string) {})
	require.NoError(t, err)
	assert.Equal(t, "标题", result["script"].(generation.Script).Title)
}
