package runner

import (
	"context"
	"fmt"
	"time"

	"manga-drama-studio/internal/domain"
)

type Executor interface {
	Execute(context.Context, domain.Node, map[string]any, func(int, string)) (map[string]any, error)
}

type MockExecutor struct{}

func (MockExecutor) Execute(ctx context.Context, node domain.Node, _ map[string]any, progress func(int, string)) (map[string]any, error) {
	steps := []struct {
		value   int
		message string
	}{{15, "准备节点"}, {48, "调用模型"}, {78, "整理输出"}, {100, "节点完成"}}
	for _, step := range steps {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(180 * time.Millisecond):
			progress(step.value, step.message)
		}
	}
	outputs := map[string]any{"summary": fmt.Sprintf("%s 已完成（演示执行器）", node.Name)}
	switch node.Type {
	case "story-input":
		outputs["story"] = node.Config["prompt"]
	case "script":
		outputs["script"] = "第 1 场：雨夜长安，古画发出低语……"
	case "character":
		outputs["character"] = "角色视觉档案与参考图集"
	case "storyboard":
		outputs["shots"] = []string{"雨夜城门", "画师回眸", "古画苏醒"}
	case "image":
		outputs["images"] = []string{"asset://shot-001.png", "asset://shot-002.png"}
	case "voice":
		outputs["audio"] = "asset://dialogue.wav"
	case "compose":
		outputs["video"] = "asset://final-demo.mp4"
	}
	return outputs, nil
}
