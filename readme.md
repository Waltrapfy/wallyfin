# Wallyfin

A lightweight media server for TV shows.  
Built in Go. Simple. Fast. No bloat.

Wallyfin is for people who just want to:
- put shows in folders
- open a clean web UI
- play episodes in the browser
- track what they watched

No giant metadata engines. No fragile reverse-proxy drama. No 10-minute library scans.

---

## Features

- 🎬 TV show library from folder structure
- 🔐 Login system with admin + normal users
- 👤 Per-user watch history
- 🖼️ Automatic poster + description scraping
- 🌍 UI language: German / English
- 📱 Mobile-friendly interface
- 🗄️ SQLite database
- 🐳 Docker support
- 📦 Single lightweight Go service

---

## Expected folder structure

```text
/media
├── Breaking Bad (2008)
│   ├── Season 01
│   │   ├── S01E01.mkv
│   │   └── S01E02.mkv
│   ├── Season 02
│   └── poster.jpg          # optional, auto-created
├── Attack on Titan (2013)
│   └── Season 01
└── Monster (2004)
    └── Season 01
```

### Recommended naming:

    Show folder: Show Name (Year)
    Episodes: S01E01...

### Quick start (Docker Compose)

```Bash

git clone https://github.com/YOUR_USER/wallyfin.git
cd wallyfin
```

Edit docker-compose.yml:

    set your media path
    set admin password

Then:

```Bash

docker compose up -d --build
```

Open:

```text
http://YOUR-SERVER-IP:8088
```

Default login (unless changed by env):

    user: admin
    password: admin ⚠️ change immediately

Quick start (Binary)

```Bash

go build -o wallyfin .
```
```Bash

WALLYFIN_MEDIA_ROOT=/path/to/shows \
WALLYFIN_PORT=8088 \
WALLYFIN_ADMIN_PASSWORD='changeme' \
./wallyfin

Configuration
Variable	Default	Description
WALLYFIN_MEDIA_ROOT	required	Path to shows
WALLYFIN_PORT	8088	HTTP port
WALLYFIN_DATA_DIR	./data	SQLite + app data
WALLYFIN_ADMIN_USER	admin	Initial admin username
WALLYFIN_ADMIN_PASSWORD	admin	Initial admin password
Metadata```

Wallyfin can automatically fetch:

    title
    description
    poster

Sources:

    TMDB if API key is set in Admin settings
    TVMaze fallback (no API key needed)

Posters are stored locally as:

```text

Show Name (Year)/poster.jpg
```

Metadata is stored as:

```text

Show Name (Year)/metadata.json
```

## FAQ
Does it transcode video?

No. It streams files directly.
Best experience with browser-friendly formats like H.264 + AAC.

Can it play MKV?

Often yes, depending on browser/codec support.
Chrome/Firefox usually handle many H.264 MKVs fine.

Where is the database?

SQLite file in the data directory, e.g.:

```text

data/wallyfin.db
```
How do I add a new show?

Just copy a folder into the media root and refresh the homepage.
Wallyfin auto-detects it and scrapes metadata if missing.
How do I change language?

Admin Panel → Settings → Language (de / en)

Is this production-ready?

It is usable as a personal/family media server.
For public internet exposure put it behind HTTPS reverse proxy and strong passwords.

Can I contribute?

Yes. PRs and issues welcome.
Roadmap

    Continue Watching row
    Mark episode watched/unwatched manually
    Better season/episode sorting
    Embedded frontend assets in binary
    Optional reverse-proxy examples (Caddy/Traefik)
    Progress timestamps (minute-level resume)

Development

```Bash
go run .
```
Requirements:

    Go 1.21+
    because there is no video transcoder built in to it, the files have to be H.264
