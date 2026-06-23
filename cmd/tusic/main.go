package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/impossibleclone/tusic-go/internal/db"
	"github.com/impossibleclone/tusic-go/internal/player"
	"github.com/impossibleclone/tusic-go/internal/ui"
)

func main() {
	database, err := db.New()
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		os.Exit(1)
	}

	mpvPlayer := player.New()
	defer mpvPlayer.Close()

	app := ui.NewAppModel(database, mpvPlayer)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Fatal UI Error: %v\n", err)
		os.Exit(1)
	}
}
