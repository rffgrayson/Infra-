package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

// ── Types ──────────────────────────────────────────────────────────────────

type LastFMImage struct {
	Text string `json:"#text"`
	Size string `json:"size"`
}

type LastFMAttr struct {
	NowPlaying string `json:"nowplaying"`
}

type LastFMTrack struct {
	Name   string `json:"name"`
	Artist struct {
		Text string `json:"#text"`
	} `json:"artist"`
	Album struct {
		Text string `json:"#text"`
	} `json:"album"`
	Image []LastFMImage `json:"image"`
	URL   string        `json:"url"`
	Attr  *LastFMAttr   `json:"@attr,omitempty"`
}

type LastFMRecentTracks struct {
	RecentTracks struct {
		Track []LastFMTrack `json:"track"`
	} `json:"recenttracks"`
}

type NowPlayingResult struct {
	IsPlaying bool   `json:"isPlaying"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	AlbumArt  string `json:"albumArt"`
	SongURL   string `json:"songUrl"`
}

// ── Handler ────────────────────────────────────────────────────────────────

func getNowPlaying(c *fiber.Ctx) error {
	apiKey   := os.Getenv("LASTFM_API_KEY")
	username := os.Getenv("LASTFM_USERNAME")

	endpoint := fmt.Sprintf(
		"https://ws.audioscrobbler.com/2.0/?method=user.getrecenttracks&user=%s&api_key=%s&format=json&limit=1",
		username, apiKey,
	)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "lastfm request failed"})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data LastFMRecentTracks
	if err := json.Unmarshal(body, &data); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to parse response"})
	}

	tracks := data.RecentTracks.Track
	if len(tracks) == 0 {
		return c.JSON(NowPlayingResult{IsPlaying: false})
	}

	track := tracks[0]
	isPlaying := track.Attr != nil && track.Attr.NowPlaying == "true"

	albumArt := ""
	for _, img := range track.Image {
		if img.Size == "extralarge" && img.Text != "" {
			albumArt = img.Text
			break
		}
	}
	if albumArt == "" {
		for _, img := range track.Image {
			if img.Size == "large" && img.Text != "" {
				albumArt = img.Text
				break
			}
		}
	}

	return c.JSON(NowPlayingResult{
		IsPlaying: isPlaying,
		Title:     track.Name,
		Artist:    track.Artist.Text,
		Album:     track.Album.Text,
		AlbumArt:  albumArt,
		SongURL:   track.URL,
	})
}

// ── Main ───────────────────────────────────────────────────────────────────

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using system env")
	}

	fmt.Println("LASTFM_API_KEY:", os.Getenv("LASTFM_API_KEY"))
	fmt.Println("LASTFM_USERNAME:", os.Getenv("LASTFM_USERNAME"))

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET",
	}))

	app.Get("/health",      func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/now-playing", getNowPlaying)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := "0.0.0.0:" + port
	fmt.Printf("portfolio-api running on %s\n", addr)
	if err := app.Listen(addr); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}