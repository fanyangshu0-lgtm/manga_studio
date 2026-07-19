package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"manga-drama-studio/internal/domain"
	"manga-drama-studio/internal/generation"
)

type ProductionExecutor struct {
	planner *generation.Planner
}

func NewProductionExecutor(planner *generation.Planner) *ProductionExecutor {
	return &ProductionExecutor{planner: planner}
}

func (e *ProductionExecutor) Execute(ctx context.Context, node domain.Node, outputs map[string]any, progress func(int, string)) (map[string]any, error) {
	switch node.Type {
	case "story-input":
		story, _ := node.Config["prompt"].(string)
		if strings.TrimSpace(story) == "" {
			return nil, errors.New("故事创意不能为空")
		}
		progress(100, "故事创意已读取")
		return map[string]any{"story": story}, nil
	case "script":
		progress(15, "正在调用 DeepSeek 编写剧本")
		story, err := outputAs[string](outputs, ".story")
		if err != nil {
			return nil, err
		}
		script, err := e.planner.Script(ctx, story)
		if err != nil {
			return nil, err
		}
		progress(100, "剧本生成完成")
		return map[string]any{"script": script}, nil
	case "character":
		progress(15, "正在生成角色视觉档案")
		script, err := outputAs[generation.Script](outputs, ".script")
		if err != nil {
			return nil, err
		}
		characters, err := e.planner.Characters(ctx, script)
		if err != nil {
			return nil, err
		}
		progress(100, "角色档案生成完成")
		return map[string]any{"character": characters}, nil
	case "storyboard":
		progress(15, "正在规划视频分镜")
		script, err := outputAs[generation.Script](outputs, ".script")
		if err != nil {
			return nil, err
		}
		characters, err := outputAs[generation.CharacterSheet](outputs, ".character")
		if err != nil {
			return nil, err
		}
		storyboard, err := e.planner.Storyboard(ctx, script, characters, node.Config)
		if err != nil {
			return nil, err
		}
		progress(100, "视频分镜规划完成")
		return map[string]any{"shots": storyboard}, nil
	default:
		return nil, fmt.Errorf("生产执行器暂不支持节点 %s", node.Type)
	}
}

func outputAs[T any](outputs map[string]any, suffix string) (T, error) {
	var zero T
	for key, value := range outputs {
		if !strings.HasSuffix(key, suffix) {
			continue
		}
		if typed, ok := value.(T); ok {
			return typed, nil
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return zero, err
		}
		if err := json.Unmarshal(encoded, &zero); err != nil {
			return zero, err
		}
		return zero, nil
	}
	return zero, fmt.Errorf("缺少上游输出 %s", suffix)
}
