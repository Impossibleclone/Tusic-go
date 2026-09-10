package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/impossibleclone/tusic-go/internal/colors"
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

type artLoadedMsg struct {
	art    string
	lyrics []LyricLine
}

func fetchArtCmd(videoID, title, artist string) tea.Cmd {
	return func() tea.Msg {
		art := getThumbnailANSI(videoID, 24, 7)
		lyrics := getLyrics(title, artist)
		if art == "" {
			art = "No Album Art"
		}
		if lyrics == nil {
			lyrics = []LyricLine{{TimeSeconds: -1, Text: "No lyrics found."}}
		}
		return artLoadedMsg{art: art, lyrics: lyrics}
	}
}
type initialMixMsg []models.Song
type searchCompleteMsg []models.Song
type streamResolvedMsg string
type radioFetchedMsg []models.Song

var (
	borderColor       = lipgloss.Color(colors.GetPywalColors().Colors["color8"])
	activeBorder      = lipgloss.Color(colors.GetPywalColors().Colors["color4"])
	baseBorderStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(borderColor)
	activeBorderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(activeBorder)
	sidebarItemStyle  = lipgloss.NewStyle().PaddingLeft(1)
	activeItemStyle   = lipgloss.NewStyle().PaddingLeft(1).Background(lipgloss.Color("#2d4b5a")).Foreground(lipgloss.Color("#B5EAD7"))
	helpDialogStyle   = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(activeBorder).Background(lipgloss.Color(colors.GetPywalColors().Special.Background)).Padding(1, 4)
)

type AppModel struct {
	db            *db.Database
	player        *player.Player
	width, height int

	searchInput textinput.Model
	searchTable table.Model
	upNextTable table.Model

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
	isResolving   bool
	loadingTicks  int
	isPaused      bool
	isLooping     bool
	upNextPending bool
	hasBooted     bool
	progressStr   string
	currentPos    float64
	totalDur      float64
	tableTitle    string
	albumArt       string
	lyrics         []LyricLine
	lyricsScroll   int
	activeLyricIdx int
	userScrolled   bool
	lastArtID      string
}

func createProgressBar(cur, dur float64, width int) string {
	if width < 4 { return "" }
	if dur <= 0 { return "[" + strings.Repeat(" ", width-2) + "]" }
	percent := cur / dur
	if percent > 1 { percent = 1 }
	filled := int(percent * float64(width-2))
	empty := width - 2 - filled
	if empty < 0 { empty = 0 }
	return "[" + strings.Repeat("█", filled) + strings.Repeat(" ", empty) + "]"
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

		mainInnerWidth := m.width - 65
		if mainInnerWidth < 10 { mainInnerWidth = 10 }
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
		for _, s := range m.searchSongs {
			rows = append(rows, table.Row{s.Title, s.Artist, s.Duration})
		}
		m.searchTable.SetRows(rows)
		m.searchTable.SetCursor(0)
		m.statusMsg = "Mix generated from Recents."

	case searchCompleteMsg:
		m.searchSongs = msg
		var rows []table.Row
		for _, s := range m.searchSongs {
			rows = append(rows, table.Row{s.Title, s.Artist, s.Duration})
		}
		m.searchTable.SetRows(rows)
		m.searchTable.SetCursor(0)
		m.activeView = ViewSearch
		m.focus = FocusTable
		m.statusMsg = "Complete."

	case radioFetchedMsg:
		m.upNextSongs = msg
		var rows []table.Row
		for _, s := range m.upNextSongs {
			rows = append(rows, table.Row{s.Title, s.Artist, s.Duration})
		}
		m.upNextTable.SetRows(rows)
		m.upNextTable.SetCursor(0)

		m.activeView = ViewUpNext
		m.tableTitle = "Up Next (Radio)"
		if len(m.upNextSongs) > 0 {
			m.upNextPending = true
		}

	case streamResolvedMsg:
		url := string(msg)
		m.isResolving = false
		if url == "" {
			m.statusMsg = "Error: Stream extraction failed. Skipping..."
			m.isLoading = false
			m.autoPlay = true
		} else {
			m.statusMsg = "Buffering stream..."
			m.isPaused = false // Reset pause flag when a new song starts
			m.isLooping = false
			m.player.Play(url)
			m.autoPlay = true
			m.loadingTicks = 0
		}

	case artLoadedMsg:
		m.albumArt = msg.art
		m.lyrics = msg.lyrics
		m.lyricsScroll = 0
		m.activeLyricIdx = 0
		m.userScrolled = false

	case tickMsg:
		cur, dur, idle, hasFile := m.player.GetProgress()

		var cmd tea.Cmd
		if m.playing != nil && m.playing.ID != m.lastArtID {
			m.lastArtID = m.playing.ID
			m.albumArt = "Loading art..."
			m.lyrics = []LyricLine{{TimeSeconds: -1, Text: "Loading lyrics..."}}
			m.lyricsScroll = 0
			m.activeLyricIdx = 0
			m.userScrolled = false
			cmd = fetchArtCmd(m.playing.ID, m.playing.Title, m.playing.Artist)
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		if m.playing != nil && len(m.lyrics) > 0 {
			for i, line := range m.lyrics {
				if line.TimeSeconds > 0 && cur >= line.TimeSeconds-1.0 {
					m.activeLyricIdx = i
				}
			}
			
			if !m.userScrolled {
				m.lyricsScroll = m.activeLyricIdx - 3
				if m.lyricsScroll < 0 {
					m.lyricsScroll = 0
				}
			}
		}

		if m.isLoading {
			m.loadingTicks++
			if m.isResolving {
				if m.loadingTicks > 0 && m.loadingTicks%2 == 0 {
					m.statusMsg = fmt.Sprintf("Extracting stream... (%ds)", m.loadingTicks)
				}
			} else {
				if m.loadingTicks > 0 && m.loadingTicks%2 == 0 {
					m.statusMsg = fmt.Sprintf("Buffering stream... (%ds)", m.loadingTicks)
				}
			}
		}

		if dur > 0 {
			m.currentPos = cur
			m.totalDur = dur
			m.progressStr = fmt.Sprintf("[%02d:%02d / %02d:%02d]", int(cur)/60, int(cur)%60, int(dur)/60, int(dur)%60)
			if m.isLoading {
				m.isLoading = false
				m.loadingTicks = 0
				m.statusMsg = "Playing."
			}
		}

		if m.isLoading && !m.isResolving && m.loadingTicks > 2 && !hasFile {
			m.isLoading = false
			m.statusMsg = "Error: Stream failed to play. Skipping..."
		}

		if idle && !m.isLoading && m.autoPlay && m.playing != nil && !m.isPaused {
			m.autoPlay = false
			cmds = append(cmds, m.playNext())
		}
		cmds = append(cmds, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }))

	case tea.MouseMsg:
		if msg.Type == tea.MouseWheelUp {
			m.userScrolled = true
			if m.lyricsScroll > 0 { m.lyricsScroll-- }
			return m, nil
		}
		if msg.Type == tea.MouseWheelDown {
			m.userScrolled = true
			m.lyricsScroll++
			return m, nil
		}
	case tea.KeyMsg:
		if m.helpOpen {
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "?" {
				m.helpOpen = false
			}
			return m, nil
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "ctrl+c":
			return m, tea.Quit
		case "?":
			m.helpOpen = true
			return m, nil
		case "/":
			m.focus = FocusSearch
			return m, nil
		case "esc":
			m.focus = FocusTable
			return m, nil
		case "[":
			m.userScrolled = true
			if m.lyricsScroll > 0 {
				m.lyricsScroll--
			}
			return m, nil
		case "]":
			m.userScrolled = true
			m.lyricsScroll++
			return m, nil
		case "h":
			if m.focus == FocusTable {
				m.focus = FocusSidebar
			}
		case "l":
			if m.focus == FocusSidebar {
				m.focus = FocusTable
			}
		case "H":
			m.activeView = ViewSearch
			m.tableTitle = "Search Results"
			return m, nil
		case "L":
			m.activeView = ViewUpNext
			m.tableTitle = "Up Next (Radio)"
			return m, nil
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
			case "j", "down":
				if m.sidebarIndex < len(m.sidebarItems)-1 {
					m.sidebarIndex++
				}
			case "k", "up":
				if m.sidebarIndex > 0 {
					m.sidebarIndex--
				}
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
			case "enter":
				cmds = append(cmds, m.playManual())
			case "p":
				m.isPaused = m.player.TogglePause()
				if m.isPaused {
					m.statusMsg = "Paused."
				} else {
					m.statusMsg = "Playing."
				}
			case "n":
				cmds = append(cmds, m.playNext())
			case "o":
				m.isLooping = m.player.ToggleLoop()
				if m.isLooping {
					m.statusMsg = "Looping."
				} else {
					m.statusMsg = "Looping."
				}
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
				if m.activeView == ViewUpNext {
					activeList = m.upNextSongs
					cursor = m.upNextTable.Cursor()
				}
				if cursor < len(activeList) {
					m.db.AddPlaylist(activeList[cursor])
					m.statusMsg = "Saved: " + activeList[cursor].Title
				}
			case "d":
				activeList := m.searchSongs
				cursor := m.searchTable.Cursor()
				if m.activeView == ViewUpNext {
					activeList = m.upNextSongs
					cursor = m.upNextTable.Cursor()
				}
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
		if m.searchTable.Cursor() >= len(m.searchSongs) {
			return nil
		}
		selected = m.searchSongs[m.searchTable.Cursor()]
	} else {
		if m.upNextTable.Cursor() >= len(m.upNextSongs) {
			return nil
		}
		selected = m.upNextSongs[m.upNextTable.Cursor()]
	}

	m.playing = &selected
	m.db.AddHistory(selected)
	m.statusMsg = "Extracting stream..."
	m.autoPlay = false
	m.isLoading = true
	m.isResolving = true
	m.loadingTicks = 0

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
			if m.searchTable.Cursor() >= len(m.searchSongs) {
				return nil
			}
			selected = m.searchSongs[m.searchTable.Cursor()]
		} else if m.isLooping {
			selected = m.searchSongs[m.searchTable.Cursor()]
		} else {
			m.upNextTable.MoveDown(1)
			if m.upNextTable.Cursor() >= len(m.upNextSongs) {
				return nil
			}
			selected = m.upNextSongs[m.upNextTable.Cursor()]
		}
	}

	m.playing = &selected
	m.db.AddHistory(selected)
	m.statusMsg = "Loading next track..."
	m.autoPlay = false
	m.isLoading = true
	m.isResolving = true
	m.loadingTicks = 0

	return func() tea.Msg { return streamResolvedMsg(ytapi.GetStreamURL(selected.ID)) }
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colors.GetPywalColors().Special.Foreground)).MarginBottom(1)

	searchBorder := baseBorderStyle
	if m.focus == FocusSearch {
		searchBorder = activeBorderStyle
	}
	// Mathematically flush top bar
	header := lipgloss.JoinHorizontal(lipgloss.Center, searchBorder.Width(m.width-16).Render(m.searchInput.View()), " ", baseBorderStyle.Width(11).Render(" ? : Help"))

	sidebarBorder := baseBorderStyle
	if m.focus == FocusSidebar {
		sidebarBorder = activeBorderStyle
	}
	var sbContent strings.Builder
	for i, item := range m.sidebarItems {
		if i == m.sidebarIndex && m.focus == FocusSidebar {
			sbContent.WriteString(activeItemStyle.Render(item) + "\n")
		} else {
			sbContent.WriteString(sidebarItemStyle.Render(item) + "\n")
		}
	}
	sidebar := sidebarBorder.Width(25).Height(m.height - 10).Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("— Library"), sbContent.String()))

	tableBorder := baseBorderStyle
	if m.focus == FocusTable {
		tableBorder = activeBorderStyle
	}

	activeTableView := m.searchTable.View()
	if m.activeView == ViewUpNext {
		activeTableView = m.upNextTable.View()
	}

	mainContent := tableBorder.Width(m.width - 63).Height(m.height - 10).Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("— "+m.tableTitle), activeTableView))
	
	var rightSidebarContent string
	if m.playing != nil && m.albumArt != "" {
		artBlock := m.albumArt
		artLines := strings.Count(artBlock, "\n")

		// Exact math: Sidebar Height (m.height - 10) minus Borders(2) minus Title(2) minus ArtLines minus padding(1)
		lyricsHeight := (m.height - 10) - 2 - 2 - artLines - 1
		if lyricsHeight < 1 {
			lyricsHeight = 1
		}

		if m.lyricsScroll > len(m.lyrics)-1 {
			m.lyricsScroll = len(m.lyrics)-1
		}
		if m.lyricsScroll < 0 {
			m.lyricsScroll = 0
		}

		var visibleLyrics strings.Builder
		for i := m.lyricsScroll; i < len(m.lyrics) && i < m.lyricsScroll+lyricsHeight; i++ {
			if i == m.activeLyricIdx {
				visibleLyrics.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render(m.lyrics[i].Text) + "\n")
			} else {
				visibleLyrics.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(m.lyrics[i].Text) + "\n")
			}
		}

		renderedLines := strings.Split(lipgloss.NewStyle().Width(28).Render(visibleLyrics.String()), "\n")
		if len(renderedLines) > lyricsHeight {
			renderedLines = renderedLines[:lyricsHeight]
		}
		lyricsBlock := strings.Join(renderedLines, "\n")
		rightSidebarContent = lipgloss.JoinVertical(lipgloss.Center, artBlock, "\n", lyricsBlock)
	} else {
		rightSidebarContent = "\n\n  Waiting for music..."
	}
	rightSidebar := baseBorderStyle.Width(30).Height(m.height - 10).Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("— Art & Lyrics [ / ]"), rightSidebarContent))

	middle := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", mainContent, " ", rightSidebar)

	nowPlaying := m.statusMsg
	if m.playing != nil {
		stateIcon := "▶ Playing"
		if m.isLoading {
			stateIcon = "⧗ Loading"
		} else if strings.HasPrefix(m.statusMsg, "Error") {
			stateIcon = "⚠ Error"
		} else if m.isPaused {
			stateIcon = "⏸ Paused"
		} else if m.isLooping {
			stateIcon = "∞ Looping"
		}
		bar := createProgressBar(m.currentPos, m.totalDur, 20)
		nowPlaying = fmt.Sprintf("%s %s %s : %s - %s", stateIcon, m.progressStr, bar, m.playing.Title, m.playing.Artist)
	}

	// Make the status stand out by using a bright floating-like bar above the player if loading or errored
	var floatingMsg string
	if m.isLoading || strings.HasPrefix(m.statusMsg, "Error") {
		floatingMsg = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#F3F99D")).
			Padding(0, 2).
			Bold(true).
			Render(" " + m.statusMsg + " ") + "\n"
	}

	footer := baseBorderStyle.Width(m.width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.MarginBottom(0).Render("— Player"), floatingMsg+lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(nowPlaying)))

	ui := lipgloss.JoinVertical(lipgloss.Left, header, middle, footer)

	if m.helpOpen {
		dialog := helpDialogStyle.Render(lipgloss.NewStyle().Bold(true).Render("Tusic Keybindings") +
			"\n\n  Navigation\n  h / l : Focus Sidebar / Songs\n  H / L : View Search / View Up Next\n  j / k : Move up / down\n\n  Playback\n  p : Play / Pause\n  n : Next Track\n\n  General\n  / : Search\n  esc / q : Close")
		ui = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog, lipgloss.WithWhitespaceChars(" "))
	}
	return ui
}
