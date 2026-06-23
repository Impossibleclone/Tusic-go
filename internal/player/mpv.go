package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dexterlb/mpvipc"
)

type Player struct {
	conn *mpvipc.Connection
	cmd  *exec.Cmd
	sock string
}

func New() *Player {
	sock := filepath.Join(os.TempDir(), "tusic-mpv.sock")
	os.Remove(sock)

	args := []string{
		"--idle=yes",
		"--no-video",
		"--force-window=no", // Ensure no blank windows pop up
		"--ytdl-format=bestaudio/best",
		fmt.Sprintf("--input-ipc-server=%s", sock),
	}

	if _, err := os.Stat("cookies.txt"); err == nil {
		args = append(args, "--ytdl-raw-options=cookies=cookies.txt")
	}

	cmd := exec.Command("mpv", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Start()

	conn := mpvipc.NewConnection(sock)
	for i := 0; i < 20; i++ {
		if err := conn.Open(); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return &Player{conn: conn, cmd: cmd, sock: sock}
}

func (p *Player) Play(streamURL string) {
	// Explicitly tell MPV to replace the current track
	p.conn.Call("loadfile", streamURL, "replace")
}

func (p *Player) Stop() {
	p.conn.Call("stop")
}

func (p *Player) TogglePause() bool {
	paused, err := p.conn.Get("pause")
	if err != nil || paused == nil {
		return false
	}
	isPaused := paused.(bool)
	p.conn.Set("pause", !isPaused)
	return !isPaused
}

func (p *Player) GetProgress() (float64, float64, bool) {
	pos, _ := p.conn.Get("time-pos")
	dur, _ := p.conn.Get("duration")
	idle, _ := p.conn.Get("idle-active")

	current, total := 0.0, 0.0
	if pos != nil {
		current = pos.(float64)
	}
	if dur != nil {
		total = dur.(float64)
	}
	isIdle := true
	if idle != nil {
		isIdle = idle.(bool)
	}

	return current, total, isIdle
}

func (p *Player) Close() {
	p.conn.Call("quit")
	p.conn.Close()
	p.cmd.Wait()
	os.Remove(p.sock)
}
