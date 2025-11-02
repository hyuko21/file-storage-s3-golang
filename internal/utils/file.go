package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type FileMetadataOutput struct {
	Streams []StreamOutput
}

type StreamOutput struct {
	Type   string `json:"codec_type"`
	Height int
	Width  int
}

func GetVideoAspectRatio(filePath string) (r string, err error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	out := &bytes.Buffer{}
	cmd.Stdout = out
	if err = cmd.Run(); err != nil {
		return
	}
	var fileMetadata FileMetadataOutput
	if err = json.Unmarshal(out.Bytes(), &fileMetadata); err != nil {
		return
	}

	var videoStream StreamOutput
	for _, v := range fileMetadata.Streams {
		if v.Type == "video" {
			videoStream = v
			break
		}
	}
	if videoStream.Height == 0 && videoStream.Width == 0 {
		return r, fmt.Errorf("file missing 'video' metadata")
	}
	ratio := float64(videoStream.Width) / float64(videoStream.Height)
	if ratio >= 1.7 && ratio <= 1.8 {
		r = "16:9"
	} else if ratio >= 0.5 && ratio <= 0.6 {
		r = "9:16"
	} else {
		r = "other"
	}
	return
}
