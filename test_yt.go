package main
import (
	"fmt"
	"github.com/kkdai/youtube/v2"
)
func main() {
	client := youtube.Client{}
	video, _ := client.GetVideo("dQw4w9WgXcQ")
	formats := video.Formats.WithAudioChannels()
	url, _ := client.GetStreamURL(video, &formats[0])
	fmt.Println(url)
}
