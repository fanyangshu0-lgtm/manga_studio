package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"manga-drama-studio/internal/assets"
	"manga-drama-studio/internal/domain"
	"manga-drama-studio/internal/generation"
	"manga-drama-studio/internal/media"
	"manga-drama-studio/internal/newapi"
	"manga-drama-studio/internal/store"
)

type ProductionExecutor struct {
	planner        *generation.Planner
	video          *newapi.Client
	assets         *assets.Service
	repository     store.Repository
	fastModel      string
	qualityModel   string
	defaultModel   string
	maxConcurrency int
	pollInterval   time.Duration
	taskTimeout    time.Duration
	composer       *media.Composer
}

func (e *ProductionExecutor) ConfigureComposer(composer *media.Composer) { e.composer = composer }

type executionInfoKey struct{}
type executionInfo struct{ RunID, ProjectID, NodeID string }

func NewProductionExecutor(planner *generation.Planner) *ProductionExecutor {
	return &ProductionExecutor{planner: planner}
}

func (e *ProductionExecutor) ConfigureVideo(client *newapi.Client, assetService *assets.Service, repository store.Repository, fastModel, qualityModel, defaultModel string, maxConcurrency int, pollInterval, taskTimeout time.Duration) {
	e.video, e.assets, e.repository = client, assetService, repository
	e.fastModel, e.qualityModel, e.defaultModel = fastModel, qualityModel, defaultModel
	e.maxConcurrency, e.pollInterval, e.taskTimeout = maxConcurrency, pollInterval, taskTimeout
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
	case "image":
		return e.generateVideos(ctx, node, outputs, progress)
	case "voice":
		progress(100, "Seedance 镜头已包含原生音频")
		return map[string]any{"audio": "seedance-native"}, nil
	case "compose":
		return e.compose(ctx, outputs, progress)
	default:
		return nil, fmt.Errorf("生产执行器暂不支持节点 %s", node.Type)
	}
}

func (e *ProductionExecutor) compose(ctx context.Context, outputs map[string]any, progress func(int, string)) (map[string]any, error) {
	if e.composer == nil || e.assets == nil || e.repository == nil {
		return nil, errors.New("FFmpeg 合成服务未配置")
	}
	info, ok := ctx.Value(executionInfoKey{}).(executionInfo)
	if !ok {
		return nil, errors.New("缺少运行上下文")
	}
	clips, err := outputAs[[]domain.Asset](outputs, ".clips")
	if err != nil {
		return nil, err
	}
	storyboard, err := outputAs[generation.Storyboard](outputs, ".shots")
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(clips))
	for index, asset := range clips {
		paths[index], err = e.assets.Path(asset)
		if err != nil {
			return nil, err
		}
	}
	cues := make([]media.Cue, len(storyboard.Shots))
	for index, shot := range storyboard.Shots {
		cues[index] = media.Cue{Text: shot.Subtitle, Duration: time.Duration(shot.DurationSeconds) * time.Second}
	}
	width, height := 1280, 720
	if len(storyboard.Shots) > 0 {
		switch storyboard.Shots[0].Ratio {
		case "9:16":
			width, height = 720, 1280
		case "1:1":
			width, height = 720, 720
		}
	}
	assetID := domain.NewID("ast")
	relative, target, err := e.assets.Target(info.ProjectID, info.RunID, info.NodeID, assetID)
	if err != nil {
		return nil, err
	}
	progress(15, "正在使用 FFmpeg 合成最终视频")
	if err := e.composer.Compose(ctx, paths, cues, target, width, height); err != nil {
		return nil, err
	}
	asset, err := e.assets.RegisterVideo(relative, target, assetID, info.ProjectID, info.RunID, info.NodeID)
	if err != nil {
		return nil, err
	}
	if err := e.repository.SaveAsset(ctx, asset); err != nil {
		return nil, err
	}
	progress(100, "最终视频合成完成")
	return map[string]any{"video": asset, "assetUrl": "/api/v1/assets/" + asset.ID}, nil
}

func (e *ProductionExecutor) generateVideos(ctx context.Context, node domain.Node, outputs map[string]any, progress func(int, string)) (map[string]any, error) {
	if e.video == nil || e.assets == nil || e.repository == nil {
		return nil, errors.New("Seedance 视频服务未配置")
	}
	info, ok := ctx.Value(executionInfoKey{}).(executionInfo)
	if !ok {
		return nil, errors.New("缺少运行上下文")
	}
	storyboard, err := outputAs[generation.Storyboard](outputs, ".shots")
	if err != nil {
		return nil, err
	}
	model := e.defaultModel
	if quality, _ := node.Config["quality"].(string); quality == "quality" {
		model = e.qualityModel
	}
	if configured, _ := node.Config["model"].(string); configured == e.fastModel || configured == e.qualityModel {
		model = configured
	}
	clips := make([]domain.Asset, len(storyboard.Shots))
	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	semaphore := make(chan struct{}, e.maxConcurrency)
	var completed atomic.Int32
	var firstErr error
	var errMu sync.Mutex
	var wg sync.WaitGroup
	for index, shot := range storyboard.Shots {
		index, shot := index, shot
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case semaphore <- struct{}{}:
			case <-workerContext.Done():
				return
			}
			defer func() { <-semaphore }()
			asset, generateErr := e.generateShot(workerContext, info, shot, model)
			if generateErr != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = generateErr
					cancel()
				}
				errMu.Unlock()
				return
			}
			clips[index] = asset
			done := completed.Add(1)
			progress(int(done)*100/len(storyboard.Shots), fmt.Sprintf("已完成 %d/%d 个视频镜头", done, len(storyboard.Shots)))
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return map[string]any{"clips": clips}, nil
}

func (e *ProductionExecutor) generateShot(ctx context.Context, info executionInfo, shot generation.Shot, model string) (domain.Asset, error) {
	run, err := e.repository.Run(ctx, info.RunID)
	if err != nil {
		return domain.Asset{}, err
	}
	state := run.VideoTasks[shot.ID]
	if state.Status == "succeeded" && state.AssetID != "" {
		if asset, assetErr := e.repository.Asset(ctx, state.AssetID); assetErr == nil && e.assets.Verify(asset) {
			return asset, nil
		}
	}
	if state.TaskID == "" {
		task, submitErr := e.video.SubmitVideo(ctx, newapi.VideoRequest{Model: model, Prompt: shot.Prompt, Seconds: strconv.Itoa(shot.DurationSeconds), Metadata: map[string]any{
			"resolution": "720p", "ratio": shot.Ratio, "watermark": false, "generate_audio": true,
		}})
		if submitErr != nil {
			return domain.Asset{}, submitErr
		}
		state = domain.VideoTaskState{ShotID: shot.ID, TaskID: task.ID, Status: task.Status}
		if err := e.saveVideoState(ctx, info.RunID, shot.ID, state); err != nil {
			return domain.Asset{}, err
		}
	}
	task, err := e.video.WaitVideo(ctx, state.TaskID, e.pollInterval, e.taskTimeout)
	if err != nil {
		state.Status, state.Error = "failed", err.Error()
		_ = e.saveVideoState(context.Background(), info.RunID, shot.ID, state)
		return domain.Asset{}, err
	}
	assetID := domain.NewID("ast")
	asset, err := e.assets.DownloadVideo(ctx, task.VideoURL, info.ProjectID, info.RunID, info.NodeID, assetID)
	if err != nil {
		return domain.Asset{}, err
	}
	if err := e.repository.SaveAsset(ctx, asset); err != nil {
		return domain.Asset{}, err
	}
	state.Status, state.AssetID, state.Error = "succeeded", asset.ID, ""
	if err := e.saveVideoState(ctx, info.RunID, shot.ID, state); err != nil {
		return domain.Asset{}, err
	}
	return asset, nil
}

func (e *ProductionExecutor) saveVideoState(ctx context.Context, runID, shotID string, state domain.VideoTaskState) error {
	_, err := e.repository.UpdateRun(ctx, runID, func(run *domain.Run) {
		if run.VideoTasks == nil {
			run.VideoTasks = map[string]domain.VideoTaskState{}
		}
		run.VideoTasks[shotID] = state
	})
	return err
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
