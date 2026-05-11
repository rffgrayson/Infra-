package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

// ── Simple cache ───────────────────────────────────────────────────────────

type cache struct {
	mu        sync.Mutex
	data      any
	fetchedAt time.Time
	ttl       time.Duration
}

func (c *cache) get() (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil || time.Since(c.fetchedAt) > c.ttl {
		return nil, false
	}
	return c.data, true
}

func (c *cache) set(data any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data  = data
	c.fetchedAt = time.Now()
}

var (
	ghCache    = &cache{ttl: 5 * time.Minute}
	wakaCache  = &cache{ttl: 5 * time.Minute}
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

type WakaLanguage struct {
	Name    string  `json:"name"`
	Text    string  `json:"text"`
	Percent float64 `json:"percent"`
}

type WakaResult struct {
	TimeCoding   string         `json:"timeCoding"`
	MainProject  string         `json:"mainProject"`
	MainEditor   string         `json:"mainEditor"`
	DailyAverage string         `json:"dailyAverage"`
	Languages    []WakaLanguage `json:"languages"`
}

// ── Helpers ────────────────────────────────────────────────────────────────

func get(url, authHeader string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	// Added User-Agent (GitHub's API strictly requires this for some endpoints)
    req.Header.Set("User-Agent", "portfolio-api")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("upstream API returned status %d", resp.StatusCode)
    }
	
	return io.ReadAll(resp.Body)
}

// ── Handlers ───────────────────────────────────────────────────────────────

func getNowPlaying(c *fiber.Ctx) error {
	apiKey   := os.Getenv("LASTFM_API_KEY")
	username := os.Getenv("LASTFM_USERNAME")

	body, err := get(fmt.Sprintf(
		"https://ws.audioscrobbler.com/2.0/?method=user.getrecenttracks&user=%s&api_key=%s&format=json&limit=1",
		username, apiKey,
	), "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "lastfm request failed"})
	}

	var data LastFMRecentTracks
	if err := json.Unmarshal(body, &data); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "parse failed"})
	}

	tracks := data.RecentTracks.Track
	if len(tracks) == 0 {
		return c.JSON(NowPlayingResult{IsPlaying: false})
	}

	track     := tracks[0]
	isPlaying := track.Attr != nil && track.Attr.NowPlaying == "true"

	albumArt := ""
	for _, img := range track.Image {
		if img.Size == "extralarge" && img.Text != "" { albumArt = img.Text; break }
	}
	if albumArt == "" {
		for _, img := range track.Image {
			if img.Size == "large" && img.Text != "" { albumArt = img.Text; break }
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

func getGitHub(c *fiber.Ctx) error {
    if cached, ok := ghCache.get(); ok {
        return c.JSON(cached)
    }

    // Pull the token from environment variables
    token := os.Getenv("GITHUB_TOKEN")
    
    authHeader := ""
    if token != "" {
        authHeader = "Bearer " + token
    }

    body, err := get(
        "https://api.github.com/users/rffgrayson/events/public",
        authHeader,
    )
    if err != nil {
        // Now this will show "upstream API returned status 401" instead of "parse failed"
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    var events []map[string]any
    if err := json.Unmarshal(body, &events); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "parse failed"})
    }

    ghCache.set(events)
    return c.JSON(events)
}

func getWakaTime(c *fiber.Ctx) error {
	if cached, ok := wakaCache.get(); ok {
		return c.JSON(cached)
	}

	apiKey  := os.Getenv("WAKATIME_API_KEY")
	encoded := base64.StdEncoding.EncodeToString([]byte(apiKey))
	body, err := get(
		"https://wakatime.com/api/v1/users/current/stats/last_7_days",
		"Basic "+encoded,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "wakatime request failed"})
	}

	var raw struct {
		Data struct {
			HumanReadableTotal        string `json:"human_readable_total"`
			HumanReadableDailyAverage string `json:"human_readable_daily_average"`
			Projects []struct {
				Name string `json:"name"`
			} `json:"projects"`
			Editors []struct {
				Name string `json:"name"`
			} `json:"editors"`
			Languages []struct {
				Name    string  `json:"name"`
				Text    string  `json:"text"`
				Percent float64 `json:"percent"`
			} `json:"languages"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "parse failed"})
	}

	d := raw.Data
	result := WakaResult{
		TimeCoding:   d.HumanReadableTotal,
		DailyAverage: d.HumanReadableDailyAverage,
	}
	if len(d.Projects) > 0 { result.MainProject = d.Projects[0].Name }
	if len(d.Editors)  > 0 { result.MainEditor  = d.Editors[0].Name  }

	for _, l := range d.Languages {
		if len(result.Languages) >= 5 { break }
		result.Languages = append(result.Languages, WakaLanguage{
			Name: l.Name, Text: l.Text, Percent: l.Percent,
		})
	}

	wakaCache.set(result)
	return c.JSON(result)
}

// ── Main ───────────────────────────────────────────────────────────────────

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: no .env file")
	}

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET",
	}))

	app.Get("/health",      func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/now-playing", getNowPlaying)
	app.Get("/github",      getGitHub)
	app.Get("/wakatime",    getWakaTime)

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	addr := "0.0.0.0:" + port
	fmt.Printf("portfolio-api running on %s\n", addr)
	if err := app.Listen(addr); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}