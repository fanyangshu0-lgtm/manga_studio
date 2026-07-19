package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Composer struct {
	FFmpeg  string
	FFprobe string
}

func NewComposer() *Composer { return &Composer{FFmpeg: "ffmpeg", FFprobe: "ffprobe"} }

func (c *Composer) Check(ctx context.Context) error {
	if _, err := c.run(ctx, c.FFmpeg, "-version"); err != nil {
		return fmt.Errorf("ffmpeg 不可用: %w", err)
	}
	if _, err := c.run(ctx, c.FFprobe, "-version"); err != nil {
		return fmt.Errorf("ffprobe 不可用: %w", err)
	}
	return nil
}

func (c *Composer) Compose(ctx context.Context, clips []string, cues []Cue, target string, width, height int) error {
	if len(clips) == 0 {
		return errors.New("没有可合成的视频镜头")
	}
	if width <= 0 || height <= 0 {
		return errors.New("输出尺寸无效")
	}
	workDir := target + ".work"
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(workDir)
	normalized := make([]string, len(clips))
	filter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,fps=30,format=yuv420p", width, height, width, height)
	for index, clip := range clips {
		normalized[index] = filepath.Join(workDir, fmt.Sprintf("clip-%03d.mp4", index+1))
		audio, err := c.hasAudio(ctx, clip)
		if err != nil {
			return err
		}
		args := []string{"-y", "-i", clip}
		if !audio {
			args = append(args, "-f", "lavfi", "-i", "anullsrc=channel_layout=stereo:sample_rate=48000", "-shortest")
		}
		args = append(args, "-vf", filter, "-c:v", "libx264", "-preset", "medium", "-c:a", "aac", "-ar", "48000", "-ac", "2", "-movflags", "+faststart", normalized[index])
		if _, err := c.run(ctx, c.FFmpeg, args...); err != nil {
			return fmt.Errorf("标准化第 %d 个镜头: %w", index+1, err)
		}
	}
	concatPath := filepath.Join(workDir, "concat.txt")
	var concat strings.Builder
	for _, clip := range normalized {
		fmt.Fprintf(&concat, "file '%s'\n", strings.ReplaceAll(filepath.ToSlash(clip), "'", "'\\''"))
	}
	if err := os.WriteFile(concatPath, []byte(concat.String()), 0o600); err != nil {
		return err
	}
	partial := target + ".part.mp4"
	_ = os.Remove(partial)
	args := []string{"-y", "-f", "concat", "-safe", "0", "-i", concatPath}
	if subtitle := BuildSRT(cues); subtitle != "" {
		subtitlePath := filepath.Join(workDir, "subtitles.srt")
		if err := os.WriteFile(subtitlePath, append([]byte{0xEF, 0xBB, 0xBF}, []byte(subtitle)...), 0o600); err != nil {
			return err
		}
		filterPath := strings.ReplaceAll(filepath.ToSlash(subtitlePath), ":", "\\:")
		filterPath = strings.ReplaceAll(filterPath, "'", "\\'")
		args = append(args, "-vf", "subtitles='"+filterPath+"':force_style='FontName=Noto Sans CJK SC,FontSize=22,Outline=2'")
	}
	args = append(args, "-c:v", "libx264", "-c:a", "aac", "-movflags", "+faststart", partial)
	if _, err := c.run(ctx, c.FFmpeg, args...); err != nil {
		_ = os.Remove(partial)
		return fmt.Errorf("合成最终视频: %w", err)
	}
	if _, err := c.run(ctx, c.FFprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", partial); err != nil {
		_ = os.Remove(partial)
		return fmt.Errorf("验证最终视频: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(partial, target)
}

func (c *Composer) hasAudio(ctx context.Context, path string) (bool, error) {
	output, err := c.run(ctx, c.FFprobe, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=index", "-of", "csv=p=0", path)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

func (c *Composer) run(ctx context.Context, executable string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, executable, args...)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if err == nil {
		return output.String(), nil
	}
	data := output.Bytes()
	if len(data) > 32<<10 {
		data = data[len(data)-(32<<10):]
	}
	exitCode := "未启动"
	if command.ProcessState != nil {
		exitCode = strconv.Itoa(command.ProcessState.ExitCode())
	}
	return "", fmt.Errorf("%s 退出失败 (%s): %s", filepath.Base(executable), exitCode, strings.TrimSpace(string(data)))
}
