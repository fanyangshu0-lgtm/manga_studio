package generation

import (
	"context"
	"encoding/json"
	"fmt"

	"manga-drama-studio/internal/newapi"
)

type JSONChatter interface {
	ChatJSON(context.Context, string, []newapi.Message, any) error
}

type Planner struct {
	client JSONChatter
	model  string
}

func NewPlanner(client JSONChatter, model string) *Planner {
	return &Planner{client: client, model: model}
}

func (p *Planner) Script(ctx context.Context, story string) (Script, error) {
	var result Script
	prompt := "请把以下创意写成适合短篇 AI 漫剧的中文结构化剧本。返回 JSON，字段严格为 title、logline、scenes；scenes 每项包含 id、location、summary、dialogue，dialogue 每项包含 character、text。创意：" + story
	if err := p.client.ChatJSON(ctx, p.model, []newapi.Message{{Role: "system", Content: "你是专业短篇漫剧编剧，只返回合法 JSON。"}, {Role: "user", Content: prompt}}, &result); err != nil {
		return Script{}, err
	}
	return result, result.Validate()
}

func (p *Planner) Characters(ctx context.Context, script Script) (CharacterSheet, error) {
	payload, _ := json.Marshal(script)
	var result CharacterSheet
	prompt := "根据剧本生成角色视觉档案。返回 JSON，字段为 characters，每个角色包含 name、role、personality、appearance；appearance 必须足够具体，供所有视频镜头保持一致。剧本：" + string(payload)
	if err := p.client.ChatJSON(ctx, p.model, []newapi.Message{{Role: "system", Content: "你是角色概念设计师，只返回合法 JSON。"}, {Role: "user", Content: prompt}}, &result); err != nil {
		return CharacterSheet{}, err
	}
	return result, result.Validate()
}

func (p *Planner) Storyboard(ctx context.Context, script Script, characters CharacterSheet, config map[string]any) (Storyboard, error) {
	scriptJSON, _ := json.Marshal(script)
	characterJSON, _ := json.Marshal(characters)
	ratio := "16:9"
	if configured, ok := config["ratio"].(string); ok && configured != "" {
		ratio = configured
	}
	prompt := fmt.Sprintf("把剧本拆成最多 %d 个连续视频分镜。返回 JSON，字段为 shots，每项包含 id、order、durationSeconds、ratio、prompt、subtitle。order 从 1 连续递增，durationSeconds 1-15，ratio 使用 %s。每个 prompt 都要重复关键角色外观、电影镜头语言、动作，并把字幕/对白明确写入提示词以便生成原生音频。剧本：%s；角色：%s", MaxShots, ratio, scriptJSON, characterJSON)
	var result Storyboard
	if err := p.client.ChatJSON(ctx, p.model, []newapi.Message{{Role: "system", Content: "你是短视频分镜导演，只返回合法 JSON。"}, {Role: "user", Content: prompt}}, &result); err != nil {
		return Storyboard{}, err
	}
	return result, result.Validate()
}
