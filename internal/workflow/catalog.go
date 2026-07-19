package workflow

type Port struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	DataType string `json:"dataType"`
	Required bool   `json:"required,omitempty"`
}

type Definition struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Color       string `json:"color"`
	Inputs      []Port `json:"inputs"`
	Outputs     []Port `json:"outputs"`
}

var definitions = []Definition{
	{Type: "story-input", Label: "故事创意", Description: "输入主题、受众和核心冲突", Category: "创作", Color: "#ef6b4a", Outputs: []Port{{ID: "story", Label: "故事", DataType: "text"}}},
	{Type: "script", Label: "剧本拆解", Description: "生成场景、对白和节奏", Category: "创作", Color: "#d95fa4", Inputs: []Port{{ID: "story", Label: "故事", DataType: "text", Required: true}}, Outputs: []Port{{ID: "script", Label: "剧本", DataType: "script"}}},
	{Type: "character", Label: "角色设定", Description: "生成角色档案和视觉参考", Category: "视觉", Color: "#8b6be8", Inputs: []Port{{ID: "script", Label: "剧本", DataType: "script", Required: true}}, Outputs: []Port{{ID: "character", Label: "角色", DataType: "character"}}},
	{Type: "storyboard", Label: "分镜规划", Description: "把剧本拆成镜头列表", Category: "视觉", Color: "#5e7be8", Inputs: []Port{{ID: "script", Label: "剧本", DataType: "script", Required: true}, {ID: "character", Label: "角色", DataType: "character", Required: true}}, Outputs: []Port{{ID: "shots", Label: "分镜", DataType: "storyboard"}}},
	{Type: "image", Label: "画面生成", Description: "按分镜批量生成一致画面", Category: "生成", Color: "#36a6a0", Inputs: []Port{{ID: "shots", Label: "分镜", DataType: "storyboard", Required: true}, {ID: "character", Label: "角色", DataType: "character", Required: true}}, Outputs: []Port{{ID: "images", Label: "画面", DataType: "image-sequence"}}},
	{Type: "voice", Label: "配音生成", Description: "按角色生成对白和旁白", Category: "生成", Color: "#d39a32", Inputs: []Port{{ID: "script", Label: "剧本", DataType: "script", Required: true}}, Outputs: []Port{{ID: "audio", Label: "音轨", DataType: "audio"}}},
	{Type: "compose", Label: "视频合成", Description: "合成画面、配音和字幕", Category: "输出", Color: "#ea4f68", Inputs: []Port{{ID: "images", Label: "画面", DataType: "image-sequence", Required: true}, {ID: "audio", Label: "音轨", DataType: "audio", Required: true}}, Outputs: []Port{{ID: "video", Label: "成片", DataType: "video"}}},
}

func Catalog() []Definition {
	result := make([]Definition, len(definitions))
	copy(result, definitions)
	return result
}

func definition(nodeType string) (Definition, bool) {
	for _, item := range definitions {
		if item.Type == nodeType {
			return item, true
		}
	}
	return Definition{}, false
}
