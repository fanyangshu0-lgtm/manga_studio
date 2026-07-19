package media

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildSRTUsesCumulativeShotTimings(t *testing.T) {
	got := BuildSRT([]Cue{{Text: "第一句", Duration: 5 * time.Second}, {Text: "第二句", Duration: 4 * time.Second}})
	assert.Equal(t, "1\n00:00:00,000 --> 00:00:05,000\n第一句\n\n2\n00:00:05,000 --> 00:00:09,000\n第二句\n\n", got)
}
