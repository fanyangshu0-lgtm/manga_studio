package workflow

import (
	"time"

	"manga-drama-studio/internal/domain"
)

func Example(projectID string) domain.Workflow {
	now := time.Now().UTC()
	nodes := []domain.Node{
		{ID: "story", Type: "story-input", Name: "赛博长安故事", Position: domain.Position{X: 40, Y: 240}, Config: map[string]any{"prompt": "在赛博长安，一名失忆画师追查会说话的古画。", "audience": "年轻用户"}},
		{ID: "script", Type: "script", Name: "剧本拆解", Position: domain.Position{X: 330, Y: 100}, Config: map[string]any{"episodes": 1, "duration": 60}},
		{ID: "character", Type: "character", Name: "角色设定", Position: domain.Position{X: 620, Y: 40}, Config: map[string]any{"style": "国潮赛博、电影光影"}},
		{ID: "storyboard", Type: "storyboard", Name: "分镜规划", Position: domain.Position{X: 620, Y: 300}, Config: map[string]any{"shotCount": 8, "ratio": "9:16"}},
		{ID: "image", Type: "image", Name: "Seedance 视频生成", Position: domain.Position{X: 920, Y: 220}, Config: map[string]any{"quality": "fast"}},
		{ID: "compose", Type: "compose", Name: "最终成片", Position: domain.Position{X: 1240, Y: 300}, Config: map[string]any{"subtitles": true}},
	}
	edges := []domain.Edge{
		{ID: "e1", Source: "story", SourceHandle: "story", Target: "script", TargetHandle: "story"},
		{ID: "e2", Source: "script", SourceHandle: "script", Target: "character", TargetHandle: "script"},
		{ID: "e3", Source: "script", SourceHandle: "script", Target: "storyboard", TargetHandle: "script"},
		{ID: "e4", Source: "character", SourceHandle: "character", Target: "storyboard", TargetHandle: "character"},
		{ID: "e5", Source: "storyboard", SourceHandle: "shots", Target: "image", TargetHandle: "shots"},
		{ID: "e6", Source: "character", SourceHandle: "character", Target: "image", TargetHandle: "character"},
		{ID: "e7", Source: "image", SourceHandle: "clips", Target: "compose", TargetHandle: "clips"},
		{ID: "e8", Source: "storyboard", SourceHandle: "shots", Target: "compose", TargetHandle: "shots"},
	}
	return domain.Workflow{ID: domain.NewID("wf"), ProjectID: projectID, Revision: 1, Nodes: nodes, Edges: edges, Viewport: domain.Viewport{X: 0, Y: 0, Zoom: 0.72}, UpdatedAt: now}
}
