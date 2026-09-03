package ytapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/impossibleclone/tusic-go/internal/models"
	"github.com/tidwall/gjson"
)

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

func Search(query string) []models.Song {
	// The "params" string is the exact code YTMusic uses to filter by "Songs"
	res := doInnerTube("search", map[string]any{
		"query":  query,
		"params": "EgWKAQIIAWoMEA4QChADEAQQCRAF",
	})

	var songs []models.Song

	sections := res.Get("contents.tabbedSearchResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents")
	sections.ForEach(func(key, section gjson.Result) bool {
		items := section.Get("musicShelfRenderer.contents")
		items.ForEach(func(k, v gjson.Result) bool {
			renderer := v.Get("musicResponsiveListItemRenderer")
			if !renderer.Exists() {
				return true
			}

			// Try the play button overlay (Standard location)
			id := renderer.Get("overlay.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchEndpoint.videoId").String()

			// Try the menu items (Fallback location)
			if id == "" {
				id = renderer.Get("menu.menuRenderer.items.0.menuNavigationItemRenderer.navigationEndpoint.watchEndpoint.videoId").String()
			}

			// Try the title's hyperlink (Fallback location)
			if id == "" {
				id = renderer.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text.runs.0.navigationEndpoint.watchEndpoint.videoId").String()
			}

			if id == "" {
				return true
			}

			title := renderer.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text.runs.0.text").String()

			runs := renderer.Get("flexColumns.1.musicResponsiveListItemFlexColumnRenderer.text.runs").Array()
			artist := "Unknown"
			duration := "Unknown"

			if len(runs) > 0 {
				artist = runs[0].Get("text").String()
			}

			// YouTube formats the subtitle runs as: Artist • Album • Duration
			if len(runs) > 4 {
				duration = runs[len(runs)-1].Get("text").String()
			} else if len(runs) == 3 {
				duration = runs[2].Get("text").String()
			}

			songs = append(songs, models.Song{
				ID:       id,
				Title:    title,
				Artist:   artist,
				Duration: duration,
			})
			return true
		})
		return true
	})

	return songs
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
			return true
		}

		title := renderer.Get("title.runs.0.text").String()
		artist := renderer.Get("longBylineText.runs.0.text").String()
		duration := renderer.Get("lengthText.runs.0.text").String()

		songs = append(songs, models.Song{ID: id, Title: title, Artist: artist, Duration: duration})
		return true
	})
	return songs
}

func GetStreamURL(videoID string) string {
	// Let mpv handle the extraction natively via ytdl:// to bypass recent YouTube 403 Forbidden blocks
	return "ytdl://" + videoID
}
