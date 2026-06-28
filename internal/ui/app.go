package ui

import (
	"fmt"
	"strings"
	"time"
	"math/rand"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/impossibleclone/tusic-go/internal/db"
	"github.com/impossibleclone/tusic-go/internal/models"
	"github.com/impossibleclone/tusic-go/internal/player"
	"github.com/impossibleclone/tusic-go/internal/ytapi"
)

type FocusMode int
type ViewMode int

const (
	FocusSearch FocusMode = iota
	FocusSidebar
	FocusTable
)

const (
	ViewSearch ViewMode = iota
	ViewUpNext
)

type tickMsg time.Time
type initialMixMsg []models.Song
type searchCompleteMsg []models.Song
type streamResolvedMsg string
type radioFetchedMsg []models.Song

var (
	borderColor       = lipgloss.Color("#4a4a4a")
	activeBorder      = lipgloss.Color("#B5EAD7")
	baseBorderStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(borderColor)
	activeBorderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(activeBorder)
	sidebarItemStyle  = lipgloss.NewStyle().PaddingLeft(1)
	activeItemStyle   = lipgloss.NewStyle().PaddingLeft(1).Background(lipgloss.Color("#2d4b5a")).Foreground(lipgloss.Color("#B5EAD7"))
	helpDialogStyle   = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(activeBorder).Background(lipgloss.Color("#000000")).Padding(1, 4)
)

type AppModel struct {
	db           *db.Database
	player       *player.Player
	width, height int

	searchInput  textinput.Model
	searchTable  table.Model
	upNextTable  table.Model

	sidebarItems []string
	sidebarIndex int

	searchSongs []models.Song
	upNextSongs []models.Song
	playing     *models.Song

	focus          FocusMode
	activeView     ViewMode 
	playingContext ViewMode 
	
	helpOpen      bool
	statusMsg     string
	autoPlay      bool
	isLoading     bool 
	isPaused      bool
	isLooping     bool
	upNextPending bool 
	hasBooted     bool
	progressStr   string
	tableTitle    string
}

func createTable() table.Model {
	t := table.New(table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	return t
}

func NewAppModel(dbase *db.Database, p *player.Player) AppModel {
	ti := textinput.New()
	ti.Placeholder = "Search Tusic..."
	ti.Focus()

	return AppModel{
		db:             dbase,
		player:         p,
		searchInput:    ti,
		searchTable:    createTable(),
		upNextTable:    createTable(),
		sidebarItems:   []string{"Made For You", "Recently Played", "My Playlist"},
		sidebarIndex:   0,
		focus:          FocusSearch,
		activeView:     ViewSearch,
		playingContext: ViewSearch,
		statusMsg:      "Nothing playing",
		autoPlay:       true,
		tableTitle:     "Made For You",
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		mainInnerWidth := m.width - 30 
		columns := []table.Column{
			{Title: "Title", Width: mainInnerWidth / 2},
			{Title: "Artist", Width: mainInnerWidth / 4},
			{Title: "Length", Width: 8},
		}
		m.searchTable.SetColumns(columns)
		m.searchTable.SetHeight(m.height - 13)
		m.upNextTable.SetColumns(columns)
		m.upNextTable.SetHeight(m.height - 13)

		if !m.hasBooted {
			m.hasBooted = true
			m.statusMsg = "Loading Made For You mix..."
			cmds = append(cmds, func() tea.Msg {
				hist := m.db.GetHistory()
				if len(hist) > 0 {
					seed := hist[rand.Intn(len(hist))]
					return searchCompleteMsg(ytapi.GetRadio(seed.ID))
				}
				return nil
			})
		}

	case initialMixMsg:
		m.searchSongs = msg
		var rows []table.Row
		for _, s := range m.searchSongs { rows = append(rows, table.Row{s.Title, s.Artist, s.Duration}) }
		m.searchTable.SetRows(rows)
		m.searchTable.SetCursor(0)
		m.statusMsg = "Mix generated from Recents."

	case searchCompleteMsg:
		m.searchSongs = msg
		var rows []table.Row
		for _, s := range m.searchSongs { rows = append(rows, table.Row{s.Title, s.Artist, s.Duration}) }
		m.searchTable.SetRows(rows)
		m.searchTable.SetCursor(0)
		m.activeView = ViewSearch
		m.focus = FocusTable
		m.statusMsg = "Complete."

	case radioFetchedMsg:
		m.upNextSongs = msg
		var rows []table.Row
		for _, s := range m.upNextSongs { rows = append(rows, table.Row{s.Title, s.Artist, s.Duration}) }
		m.upNextTable.SetRows(rows)
		m.upNextTable.SetCursor(0)
		
		m.activeView = ViewUpNext
		m.tableTitle = "Up Next (Radio)"
		if len(m.upNextSongs) > 0 {
			m.upNextPending = true 
		}

	case streamResolvedMsg:
		url := string(msg)
		if url == "" {
			m.statusMsg = "Error: Stream extraction failed."
			m.isLoading = false 
			m.autoPlay = false
		} else {
			m.statusMsg = "Playing."
			m.isPaused = false // Reset pause flag when a new song starts
			m.isLooping = false
			m.player.Play(url)
			m.autoPlay = true
		}

	case tickMsg:
		cur, dur, idle := m.player.GetProgress()
		
		if dur > 0 {
			m.progressStr = fmt.Sprintf("[%02d:%02d / %02d:%02d]", int(cur)/60, int(cur)%60, int(dur)/60, int(dur)%60)
			m.isLoading = false
		}

		if idle && !m.isLoading && m.autoPlay && m.playing != nil {
			m.autoPlay = false
			cmds = append(cmds, m.playNext())
		}
		cmds = append(cmds, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }))

	case tea.KeyMsg:
		if m.helpOpen {
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "?" { m.helpOpen = false }
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c": return m, tea.Quit
		case "?": m.helpOpen = true; return m, nil
		case "/": m.focus = FocusSearch; return m, nil
		case "esc": m.focus = FocusTable; return m, nil
		case "h": if m.focus == FocusTable { m.focus = FocusSidebar }
		case "l": if m.focus == FocusSidebar { m.focus = FocusTable }
		case "H": m.activeView = ViewSearch; m.tableTitle = "Search Results"; return m, nil
		case "L": m.activeView = ViewUpNext; m.tableTitle = "Up Next (Radio)"; return m, nil
		}

		if m.focus == FocusSearch {
			if msg.String() == "enter" && m.searchInput.Value() != "" {
				query := m.searchInput.Value()
				m.tableTitle = "Search: " + query
				m.statusMsg = "Searching for: " + query + "..."
				m.searchInput.SetValue("")
				cmds = append(cmds, func() tea.Msg { return searchCompleteMsg(ytapi.Search(query + "song")) })
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)

		} else if m.focus == FocusSidebar {
			switch msg.String() {
			case "j", "down": if m.sidebarIndex < len(m.sidebarItems)-1 { m.sidebarIndex++ }
			case "k", "up": if m.sidebarIndex > 0 { m.sidebarIndex-- }
			case "enter":
				selection := m.sidebarItems[m.sidebarIndex]
				m.tableTitle = selection
				m.activeView = ViewSearch
				if selection == "Recently Played" {
					cmds = append(cmds, func() tea.Msg { return searchCompleteMsg(m.db.GetHistory()) })
				} else if selection == "My Playlist" {
					cmds = append(cmds, func() tea.Msg { return searchCompleteMsg(m.db.GetPlaylist()) })
				}
			}

		} else if m.focus == FocusTable {
			switch msg.String() {
			case "enter": cmds = append(cmds, m.playManual())
			case "p":
				m.isPaused = m.player.TogglePause()
				if m.isPaused { m.statusMsg = "Paused." } else { m.statusMsg = "Playing." }
			case "n": cmds = append(cmds, m.playNext())
			case "o":
				m.isLooping = m.player.ToggleLoop()
				if m.isLooping{ m.statusMsg = "Looping." } else { m.statusMsg = "Looping." }
			case "r":
				hist := m.db.GetHistory()
				if len(hist) > 0 {
					m.statusMsg = "Refreshing your mix..."
					m.tableTitle = "Made For You"
					cmds = append(cmds, func() tea.Msg {
						seed := hist[rand.Intn(len(hist))]
						return searchCompleteMsg(ytapi.GetRadio(seed.ID)) 
					})
				} else {
					m.statusMsg = "Play a song first to generate a mix!"
				}
			case "s":
				activeList := m.searchSongs
				cursor := m.searchTable.Cursor()
				if m.activeView == ViewUpNext { activeList = m.upNextSongs; cursor = m.upNextTable.Cursor() }
				if cursor < len(activeList) {
					m.db.AddPlaylist(activeList[cursor])
					m.statusMsg = "Saved: " + activeList[cursor].Title
				}
			case "d":
				activeList := m.searchSongs
				cursor := m.searchTable.Cursor()
				if m.activeView == ViewUpNext { activeList = m.upNextSongs; cursor = m.upNextTable.Cursor() }
				if cursor < len(activeList) {
					m.db.RemoveSongCompletely(activeList[cursor].ID)
					m.statusMsg = "Removed: " + activeList[cursor].Title
				}
			}

			if m.activeView == ViewSearch {
				m.searchTable, cmd = m.searchTable.Update(msg)
			} else {
				m.upNextTable, cmd = m.upNextTable.Update(msg)
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) playManual() tea.Cmd {
	m.playingContext = m.activeView
	m.upNextPending = false 
	var selected models.Song

	if m.playingContext == ViewSearch {
		if m.searchTable.Cursor() >= len(m.searchSongs) { return nil }
		selected = m.searchSongs[m.searchTable.Cursor()]
	} else {
		if m.upNextTable.Cursor() >= len(m.upNextSongs) { return nil }
		selected = m.upNextSongs[m.upNextTable.Cursor()]
	}

	m.playing = &selected
	m.db.AddHistory(selected)
	m.statusMsg = "Extracting stream..."
	m.autoPlay = false
	m.isLoading = true

	cmds := []tea.Cmd{
		func() tea.Msg { return streamResolvedMsg(ytapi.GetStreamURL(selected.ID)) },
	}

	if m.playingContext == ViewSearch {
		m.statusMsg = "Extracting stream & generating mix..."
		cmds = append(cmds, func() tea.Msg { return radioFetchedMsg(ytapi.GetRadio(selected.ID)) })
	}

	return tea.Batch(cmds...)
}

func (m *AppModel) playNext() tea.Cmd {
	var selected models.Song

	if m.upNextPending && len(m.upNextSongs) > 0 {
		m.playingContext = ViewUpNext
		m.upNextPending = false
		m.upNextTable.SetCursor(0) 
		selected = m.upNextSongs[0]
	} else {
		if m.playingContext == ViewSearch {
			m.searchTable.MoveDown(1)
			if m.searchTable.Cursor() >= len(m.searchSongs) { return nil }
			selected = m.searchSongs[m.searchTable.Cursor()]
		} else if m.isLooping {
			selected = m.searchSongs[m.searchTable.Cursor()]
		} else {
			m.upNextTable.MoveDown(1)
			if m.upNextTable.Cursor() >= len(m.upNextSongs) { return nil }
			selected = m.upNextSongs[m.upNextTable.Cursor()]
		}
	}

	m.playing = &selected
	m.db.AddHistory(selected)
	m.statusMsg = "Loading next track..."
	m.autoPlay = false
	m.isLoading = true

	return func() tea.Msg { return streamResolvedMsg(ytapi.GetStreamURL(selected.ID)) }
}

func (m AppModel) View() string {
	if m.width == 0 { return "Initializing..." }
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241")).MarginBottom(1)

	searchBorder := baseBorderStyle
	if m.focus == FocusSearch { searchBorder = activeBorderStyle }
	// Mathematically flush top bar
	header := lipgloss.JoinHorizontal(lipgloss.Center, searchBorder.Width(m.width-16).Render(m.searchInput.View()), " ", baseBorderStyle.Width(11).Render(" ? : Help"))

	sidebarBorder := baseBorderStyle
	if m.focus == FocusSidebar { sidebarBorder = activeBorderStyle }
	var sbContent strings.Builder
	for i, item := range m.sidebarItems {
		if i == m.sidebarIndex && m.focus == FocusSidebar {
			sbContent.WriteString(activeItemStyle.Render(item) + "\n")
		} else {
			sbContent.WriteString(sidebarItemStyle.Render(item) + "\n")
		}
	}
	sidebar := sidebarBorder.Width(25).Height(m.height-10).Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("— Library"), sbContent.String()))

	tableBorder := baseBorderStyle
	if m.focus == FocusTable { tableBorder = activeBorderStyle }
	
	activeTableView := m.searchTable.View()
	if m.activeView == ViewUpNext { activeTableView = m.upNextTable.View() }

	// Fixed mathematical width so the right border no longer clips outside the terminal
	mainContent := tableBorder.Width(m.width-30).Height(m.height-10).Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("— "+m.tableTitle), activeTableView))
	middle := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", mainContent)

	nowPlaying := m.statusMsg
	if m.playing != nil { 
		// Now actively reads the isPaused flag so the UI updates correctly
		stateIcon := "▶ Playing"
		if m.isLooping { stateIcon = "∞ Looping" }
		if m.isPaused { stateIcon = "⏸ Paused" }
		nowPlaying = fmt.Sprintf("%s %s : %s - %s", stateIcon, m.progressStr, m.playing.Title, m.playing.Artist) 
	}
	
	footer := baseBorderStyle.Width(m.width-2).Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.MarginBottom(0).Render("— Player"), lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(nowPlaying)))

	ui := lipgloss.JoinVertical(lipgloss.Left, header, middle, footer)

	if m.helpOpen {
		dialog := helpDialogStyle.Render(lipgloss.NewStyle().Bold(true).Render("Tusic Keybindings") + 
		"\n\n  Navigation\n  h / l : Focus Sidebar / Songs\n  H / L : View Search / View Up Next\n  j / k : Move up / down\n\n  Playback\n  p : Play / Pause\n  n : Next Track\n\n  General\n  / : Search\n  esc / q : Close")
		ui = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog, lipgloss.WithWhitespaceChars(" "))
	}
	return ui
}
