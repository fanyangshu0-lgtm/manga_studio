package generation

import (
	"errors"
	"fmt"
	"strings"
)

const MaxShots = 12

type Dialogue struct {
	Character string `json:"character"`
	Text      string `json:"text"`
}

type Scene struct {
	ID       string     `json:"id"`
	Location string     `json:"location"`
	Summary  string     `json:"summary"`
	Dialogue []Dialogue `json:"dialogue"`
}

type Script struct {
	Title   string  `json:"title"`
	Logline string  `json:"logline"`
	Scenes  []Scene `json:"scenes"`
}

func (s Script) Validate() error {
	if strings.TrimSpace(s.Title) == "" || strings.TrimSpace(s.Logline) == "" || len(s.Scenes) == 0 {
		return errors.New("剧本必须包含标题、梗概和场景")
	}
	for _, scene := range s.Scenes {
		if strings.TrimSpace(scene.ID) == "" || strings.TrimSpace(scene.Summary) == "" {
			return errors.New("每个场景必须包含 ID 和内容")
		}
	}
	return nil
}

type Character struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Personality string `json:"personality"`
	Appearance  string `json:"appearance"`
}

type CharacterSheet struct {
	Characters []Character `json:"characters"`
}

func (s CharacterSheet) Validate() error {
	if len(s.Characters) == 0 {
		return errors.New("角色表不能为空")
	}
	for _, character := range s.Characters {
		if strings.TrimSpace(character.Name) == "" || strings.TrimSpace(character.Appearance) == "" {
			return errors.New("角色必须包含姓名和外观")
		}
	}
	return nil
}

type Shot struct {
	ID              string `json:"id"`
	Order           int    `json:"order"`
	DurationSeconds int    `json:"durationSeconds"`
	Ratio           string `json:"ratio"`
	Prompt          string `json:"prompt"`
	Subtitle        string `json:"subtitle"`
}

type Storyboard struct {
	Shots []Shot `json:"shots"`
}

func (s Storyboard) Validate() error {
	if len(s.Shots) == 0 || len(s.Shots) > MaxShots {
		return fmt.Errorf("分镜数量必须在 1 到 %d 之间", MaxShots)
	}
	seen := map[string]bool{}
	allowedRatios := map[string]bool{"16:9": true, "9:16": true, "1:1": true}
	for index, shot := range s.Shots {
		if strings.TrimSpace(shot.ID) == "" || seen[shot.ID] {
			return errors.New("分镜 ID 必须存在且唯一")
		}
		seen[shot.ID] = true
		if shot.Order != index+1 {
			return errors.New("分镜顺序必须从 1 连续递增")
		}
		if shot.DurationSeconds < 1 || shot.DurationSeconds > 15 {
			return errors.New("单个分镜时长必须在 1 到 15 秒之间")
		}
		if !allowedRatios[shot.Ratio] {
			return errors.New("分镜画幅仅支持 16:9、9:16 或 1:1")
		}
		if strings.TrimSpace(shot.Prompt) == "" || strings.TrimSpace(shot.Subtitle) == "" {
			return errors.New("分镜提示词和字幕不能为空")
		}
	}
	return nil
}
