package colors

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PywalColors struct {
	Special struct {
		Background string `json:"background"`
		Foreground string `json:"foreground"`
		Cursor     string `json:"cursor"`
	} `json:"special"`
	Colors map[string]string `json:"colors"`
}

const defaultColors = `{
	"special": {
		"background": "#0c101b",
		"foreground": "#c2c3c6",
		"cursor": "#c2c3c6"
	},
	"colors": {
		"color0": "#0c101b",
		"color1": "#294245",
		"color2": "#3f4c42",
		"color3": "#50714f",
		"color4": "#6f7b54",
		"color5": "#505464",
		"color6": "#5a8464",
		"color7": "#8f9299",
		"color8": "#5b5f6f",
		"color9": "#37595C",
		"color10": "#546659",
		"color11": "#6B976A",
		"color12": "#95A570",
		"color13": "#6B7086",
		"color14": "#79B186",
		"color15": "#c2c3c6"
	}
}`

func GetPywalColors() PywalColors {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	walPath := filepath.Join(home, ".cache", "wal", "colors.json")

	file, err := os.ReadFile(walPath)
	if err != nil {
		file = []byte(defaultColors)
	}

	var theme PywalColors
	if err := json.Unmarshal(file, &theme); err != nil {
		panic(err)
	}
	return theme
}
