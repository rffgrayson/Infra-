# infra-

Lightweight Go API powering [aryff.com](https://aryff.com). Aggregates data from Last.fm, WakaTime, and GitHub — serving it with caching to the portfolio frontend.

Built with [Fiber](https://gofiber.io), deployed on [Railway](https://railway.app).

---

## Endpoints

| Method | Route | Description |
|--------|-------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/now-playing` | Currently playing or last played track from Last.fm |
| `GET` | `/wakatime` | Coding stats for the last 7 days from WakaTime |
| `GET` | `/github` | Recent public GitHub events (cached 5 min) |

---

## Stack

- **Go** — core language
- **Fiber v2** — HTTP framework
- **godotenv** — environment variable loading
- **Last.fm API** — now playing / recently played
- **WakaTime API** — coding activity
- **GitHub API** — public events
- **Railway** — deployment

---

## Local Development

**Prerequisites:** Go 1.22+

```bash
git clone https://github.com/rffgrayson/infra-
cd infra-
```

Create a `.env` file:

```env
LASTFM_API_KEY=your_lastfm_api_key
LASTFM_USERNAME=your_lastfm_username
WAKATIME_API_KEY=your_wakatime_api_key
PORT=8080
```

Run:

```bash
go mod tidy
go run main.go
```

Test:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/now-playing
curl http://localhost:8080/wakatime
curl http://localhost:8080/github
```

---

## Deployment

Deployed on Railway with environment variables set via the Railway dashboard. Auto-deploys on push to `main`.

---

## Response Examples

**`/now-playing`**
```json
{
  "isPlaying": true,
  "title": "vulnerable",
  "artist": "dhruv",
  "album": "rapunzel",
  "albumArt": "https://lastfm.freetls.fastly.net/...",
  "songUrl": "https://www.last.fm/music/dhruv/_/vulnerable"
}
```

**`/wakatime`**
```json
{
  "timeCoding": "1 hr 2 mins",
  "mainProject": "portfolio",
  "mainEditor": "VS Code",
  "dailyAverage": "20 mins",
  "languages": [
    { "name": "TypeScript", "text": "52 mins", "percent": 83.54 },
    { "name": "Go", "text": "3 mins", "percent": 5.21 }
  ]
}
```

---

## Related

- [portfolio](https://github.com/rffgrayson/portfolio) — the Next.js frontend that consumes this API