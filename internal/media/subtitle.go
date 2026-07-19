package media

import (
	"fmt"
	"strings"
	"time"
)

type Cue struct {
	Text     string
	Duration time.Duration
}

func BuildSRT(cues []Cue) string {
	var result strings.Builder
	start := time.Duration(0)
	index := 1
	for _, cue := range cues {
		text := strings.TrimSpace(strings.ReplaceAll(cue.Text, "\r\n", "\n"))
		if text == "" || cue.Duration <= 0 {
			start += cue.Duration
			continue
		}
		end := start + cue.Duration
		fmt.Fprintf(&result, "%d\n%s --> %s\n%s\n\n", index, srtTime(start), srtTime(end), text)
		start, index = end, index+1
	}
	return result.String()
}

func srtTime(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	totalMilliseconds := value.Milliseconds()
	hours := totalMilliseconds / 3_600_000
	minutes := totalMilliseconds / 60_000 % 60
	seconds := totalMilliseconds / 1_000 % 60
	milliseconds := totalMilliseconds % 1_000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}
