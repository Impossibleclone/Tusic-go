package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/impossibleclone/tusic-go/internal/models"
)

type Database struct {
	conn *sql.DB
}

func New() (*Database, error) {
	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, ".local", "share", "tusic")
	os.MkdirAll(dbDir, os.ModePerm)

	dbPath := filepath.Join(dbDir, "tusic.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &Database{conn: conn}
	db.setupTables()
	return db, nil
}

func (db *Database) setupTables() {
	history := `CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		video_id TEXT, title TEXT, artist TEXT, duration TEXT,
		played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	playlist := `CREATE TABLE IF NOT EXISTS playlist (
		video_id TEXT PRIMARY KEY, title TEXT, artist TEXT, duration TEXT
	);`
	db.conn.Exec(history)
	db.conn.Exec(playlist)
}

func (db *Database) AddHistory(s models.Song) {
	db.conn.Exec("INSERT INTO history (video_id, title, artist, duration) VALUES (?, ?, ?, ?)", s.ID, s.Title, s.Artist, s.Duration)
}

func (db *Database) GetHistory() []models.Song {
	rows, err := db.conn.Query("SELECT video_id, title, artist, duration FROM history ORDER BY played_at DESC LIMIT 50")
	if err != nil {
		return []models.Song{}
	}
	defer rows.Close()
	return scanSongs(rows)
}

func (db *Database) AddPlaylist(s models.Song) {
	db.conn.Exec("INSERT OR IGNORE INTO playlist (video_id, title, artist, duration) VALUES (?, ?, ?, ?)", s.ID, s.Title, s.Artist, s.Duration)
}

func (db *Database) GetPlaylist() []models.Song {
	rows, err := db.conn.Query("SELECT video_id, title, artist, duration FROM playlist")
	if err != nil {
		return []models.Song{}
	}
	defer rows.Close()
	return scanSongs(rows)
}

func (db *Database) RemoveSongCompletely(videoID string) bool {
	cleanID := strings.TrimSpace(videoID)
	
	res1, _ := db.conn.Exec("DELETE FROM playlist WHERE video_id = ?", cleanID)
	pDel, _ := res1.RowsAffected()
	
	res2, _ := db.conn.Exec("DELETE FROM history WHERE video_id = ?", cleanID)
	hDel, _ := res2.RowsAffected()
	
	return (pDel + hDel) > 0
}

func scanSongs(rows *sql.Rows) []models.Song {
	var songs []models.Song
	for rows.Next() {
		var s models.Song
		rows.Scan(&s.ID, &s.Title, &s.Artist, &s.Duration)
		songs = append(songs, s)
	}
	return songs
}
