package ytapi

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"net/http"
	"github.com/tidwall/gjson"
	"io"
	"encoding/json"

	"github.com/impossibleclone/tusic-go/internal/models"
)

func fetchFast(baseArgs []string) []models.Song {
	args := append([]string{"--print", "%(id)s|||%(title)s|||%(channel)s|||%(duration_string)s", "--flat-playlist", "--no-warnings", "--quiet"}, baseArgs...)

	if _, err := os.Stat("cookies.txt"); err == nil {
		args = append([]string{"--cookies", "cookies.txt"}, args...)
	}

	cmd := exec.Command("yt-dlp", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Run()

	var songs []models.Song
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "|||")
		if len(parts) >= 4 {
			dur := parts[3]
			if dur == "NA" { dur = "Unknown" }
			songs = append(songs, models.Song{
				ID:       parts[0],
				Title:    parts[1],
				Artist:   parts[2],
				Duration: dur,
			})
		}
	}
	return songs
}

func Search(query string) []models.Song {
	// ytsearch20 limits it to 20 results for instant loading
	return fetchFast([]string{fmt.Sprintf("ytsearch20:%s", query)})
}

func doInnerTube(endpoint string, payload map[string]any) gjson.Result {
	url := "https://music.youtube.com/youtubei/v1/" + endpoint
	payload["context"] = map[string]any{
		"client": map[string]any{
			"clientName":    "WEB_REMIX",
			"clientVersion": "1.20230508.00.00",
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return gjson.Result{}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return gjson.ParseBytes(respBody)
}

func GetRadio(videoID string) []models.Song {
	res := doInnerTube("next", map[string]any{"playlistId": "RDAMVM" + videoID})
	var songs []models.Song

	items := res.Get("contents.singleColumnMusicWatchNextResultsRenderer.tabbedRenderer.watchNextTabbedResultsRenderer.tabs.0.tabRenderer.content.musicQueueRenderer.content.playlistPanelRenderer.contents")

	items.ForEach(func(key, value gjson.Result) bool {
		renderer := value.Get("playlistPanelVideoRenderer")
		if !renderer.Exists() {
			return true
		}

		id := renderer.Get("videoId").String()
		if id == "" || id == videoID {
			return true // Skip missing items or the seed song itself
		}

		title := renderer.Get("title.runs.0.text").String()
		artist := renderer.Get("longBylineText.runs.0.text").String()
		duration := renderer.Get("lengthText.runs.0.text").String()

		songs = append(songs, models.Song{
			ID:       id,
			Title:    title,
			Artist:   artist,
			Duration: duration,
		})
		return true
	})

	return songs
}

func GetStreamURL(videoID string) string {
	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID) + "&sp=EgIQAQ%253D%253D"
	args := []string{"-g", "-f", "bestaudio/best", "--no-warnings", "--quiet"}

	if _, err := os.Stat("cookies.txt"); err == nil {
		args = append([]string{"--cookies", "cookies.txt"}, args...)
	}
	args = append(args, url)

	cmd := exec.Command("yt-dlp", args...)
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			cleanUrl := strings.TrimSpace(lines[len(lines)-1])
			if strings.HasPrefix(cleanUrl, "http") {
				return cleanUrl
			}
		}
	}
	return ""
}
