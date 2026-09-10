package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

type LyricLine struct {
	TimeSeconds float64
	Text        string
}

type lrcResponse struct {
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

var lrcRegex = regexp.MustCompile(`\[(\d{2}):(\d{2}\.\d{2,})\](.*)`)

func getLyrics(title, artist string) []LyricLine {
	query := fmt.Sprintf("https://lrclib.net/api/search?track_name=%s&artist_name=%s", url.QueryEscape(title), url.QueryEscape(artist))
	resp, err := http.Get(query)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var res []lrcResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || len(res) == 0 {
		return nil
	}
	
	rawLrc := res[0].SyncedLyrics
	if rawLrc == "" {
		rawLrc = res[0].PlainLyrics
		lines := strings.Split(rawLrc, "\n")
		var out []LyricLine
		for _, l := range lines {
			out = append(out, LyricLine{TimeSeconds: -1, Text: l})
		}
		return out
	}

	var out []LyricLine
	for _, line := range strings.Split(rawLrc, "\n") {
		matches := lrcRegex.FindStringSubmatch(line)
		if len(matches) == 4 {
			min, _ := strconv.ParseFloat(matches[1], 64)
			sec, _ := strconv.ParseFloat(matches[2], 64)
			text := strings.TrimSpace(matches[3])
			out = append(out, LyricLine{TimeSeconds: min*60 + sec, Text: text})
		}
	}
	return out
}

func rgbANSI(c color.Color, bg bool) string {
	r, g, b, _ := c.RGBA()
	prefix := "38"
	if bg {
		prefix = "48"
	}
	return fmt.Sprintf("\x1b[%s;2;%d;%d;%dm", prefix, r>>8, g>>8, b>>8)
}

func getThumbnailANSI(videoID string, termWidth, termHeight int) string {
	resp, err := http.Get("https://i.ytimg.com/vi/" + videoID + "/hqdefault.jpg")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	cropped := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(image.Rect(0, 45, bounds.Max.X, bounds.Max.Y-45))

	dst := image.NewRGBA(image.Rect(0, 0, termWidth, termHeight*2))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Src, nil)

	var sb strings.Builder
	for y := 0; y < termHeight; y++ {
		for x := 0; x < termWidth; x++ {
			top := dst.At(x, y*2)
			bottom := dst.At(x, y*2+1)
			sb.WriteString(rgbANSI(top, false))
			sb.WriteString(rgbANSI(bottom, true))
			sb.WriteString("▀")
		}
		sb.WriteString("\x1b[0m\n") // Reset and newline
	}
	return sb.String()
}
