# MangaHub

MangaHub is a Go-based backend for managing a manga library and synchronizing user progress in real time.

It combines multiple communication styles:
- **HTTP (Gin)** for auth + manga search + user library/progress
- **TCP** for real-time progress broadcasts
- **UDP** for lightweight notification broadcasts (subscribe + push)
- **gRPC** for internal service calls
- **WebSocket** endpoint for chat-style real-time messages

## Result / What you get
- Persistent data in **SQLite** (`users`, `manga`, `user_progress`)
- JWT-based auth (`/auth/register`, `/auth/login`, protected `/users/*`)
- Manga search APIs (simple and advanced)
- Real-time progress sync via **TCP broadcast** when a user updates progress
- A UDP notification demo endpoint (`/admin/notify`)
- WebSocket endpoint (`/ws`) for real-time message broadcasting
- Docker and docker-compose setup for containerized deployment

## How to run (local)

### 1. Requirements
- Go 1.21+

### 2. Start the server
From the repo root:

```bash
go run ./cmd/api-server/main.go
```

Default ports used by the server:
- HTTP: **http://localhost:8080**
- TCP sync: **:9090**
- UDP notifications: **:9091/udp**
- gRPC: **:9092**

### 3. Environment variables
- `DB_PATH` (optional): SQLite file path. If unset, the server uses `./data.db`.
- `JWT_SECRET` (optional): JWT signing secret. If unset, a default secret is used.

Example:

```bash
set DB_PATH=./data.db
set JWT_SECRET=your_secret_here

go run ./cmd/api-server/main.go
```

## Run with Docker Compose

```bash
docker compose up --build
```

Compose exposes:
- `8080` (HTTP)
- `9090` (TCP)
- `9091/udp` (UDP)
- `9092` (gRPC)

> Note: The container uses `/app/data` volume mount, while the server defaults to `./data.db` unless `DB_PATH` is set. If you want DB persistence to land in the mounted volume, set `DB_PATH` to something like `/app/data/data.db`.

## API usage

### Health check
- `GET /health`

Example:
```bash
curl http://localhost:8080/health
```

### Auth (JWT)

#### Register
- `POST /auth/register`

Body:
```json
{ "username": "alice", "password": "secret123" }
```

#### Login
- `POST /auth/login`

Body:
```json
{ "username": "admin", "password": "123456" }
```

Response contains:
- `token`

#### Use JWT for protected endpoints
Send:
```
Authorization: Bearer <token>
```

### Manga search

#### Search by query string
- `GET /manga?query=<text>`

Example:
```bash
curl "http://localhost:8080/manga?query=naruto"
```

#### Get a manga by id
- `GET /manga/:id`

Example:
```bash
curl http://localhost:8080/manga/one-piece
```

#### Advanced search (JSON)
- `POST /manga/search`

Body shape:
```json
{
  "keyword": "one",
  "genres": ["action"],
  "status": "",
  "year_range": [1900, 2020],
  "rating": 8.5,
  "sort_by": "popularity"
}
```

### User library & progress (protected)
All endpoints require `Authorization: Bearer <token>`.

#### Add manga to library
- `POST /users/library`

Body:
```json
{ "manga_id": "<id>", "status": "reading" }
```

#### Get user library
- `GET /users/library`

#### Update progress
- `PUT /users/progress`

Body:
```json
{ "manga_id": "<id>", "current_chapter": 12 }
```

**Behavior:** updating progress triggers a **TCP broadcast** to connected TCP clients.

## WebSocket

- `GET /ws`

Web client can connect to:
- `ws://localhost:8080/ws`

The frontend included in `frontend/` connects automatically after login.

## TCP sync (high level)
- TCP server listens on **:9090**
- It broadcasts JSON messages with user progress updates to all connected TCP clients.

## UDP notifications (subscribe + broadcast)
- UDP server listens on **:9091/udp**

### Subscribe
A UDP client should send the string:
- `subscribe`

### Broadcast
A notification can be triggered via:
- `POST /admin/notify`

Body (example):
```json
{
  "type": "update",
  "manga_id": "<id>",
  "message": "Manga updated",
  "timestamp": 1710000000
}
```

This sends the payload to all subscribed UDP clients.

## Project structure (quick map)
- `cmd/api-server/main.go`: server startup + route registration
- `internal/auth/*`: register/login + JWT middleware
- `internal/manga/handler.go`: manga search endpoints
- `internal/user/handler.go`: library + progress endpoints (TCP broadcast)
- `internal/tcp/*`: TCP sync server
- `internal/udp/*`: UDP notification server
- `internal/websocket/*`: WebSocket chat hub
- `internal/grpc/server.go`: internal gRPC implementation
- `pkg/database/db.go`: SQLite schema + migrations-on-start
- `frontend/`: simple dashboard UI

## Notes
- `./config.yaml` exists in the repo, but the running server primarily uses env vars (`DB_PATH`, `JWT_SECRET`).
- Default JWT secret is used only if `JWT_SECRET` is not set.