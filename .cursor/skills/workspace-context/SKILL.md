---
name: serj-tubin-workspace-context
description: Provides context about the serj-tubin personal website ecosystem (backend, Vue frontend, gymstats React app). Use when working on serj-tubin-com, serj-tubin-vue, or gymstats features, deployment, or cross-project integration.
---

# Serj-Tubin Workspace Context

This skill gives the agent shared context for the three related personal projects that form the serj-tubin.com site and utilities.

## The Three Projects

| Project | Tech | Purpose |
|--------|------|---------|
| **serj-tubin-com** | Go (Golang) | Backend API and services |
| **serj-tubin-vue** | Vue.js (Vuetify) | Main website frontend |
| **gymstats** | React (Vite, Chart.js) | Gym exercise analytics app (main gymstats experience) |

Gymstats remains in React. The Vue site has a `/gymstats` section (Log, Stats, List) for quick logging and list view; the React app is the primary gymstats experience (analytics, progress charts, progression rate, date-range exercises) and will be deployed at **gymstats.serj-tubin.com** when ready. Not deployed remotely yet.

## Backend (serj-tubin-com)

- **Stack**: Go, PostgreSQL, Redis, Gorilla Mux.
- **Modules** (from `internal/`): `auth`, `blog`, `file_box`, `geoip`, `gymstats` (events + exercises), `misc` (quotes), `netlog` (internet activity), `notes_box`, `spotify`, `visitor_board`, `weather`, middleware, telemetry (Prometheus, Honeycomb, Sentry).
- **Key routes**: Auth (`/a/login`), blog, netlog, gymstats (`/gymstats/...`), notes, visitor board, Spotify, weather, misc. File box is a **separate service** (see Deployment).
- **Config**: `config.toml` (sections: dockerdev, development, production). DB name: `serj_blogs`. Production backend listens on `localhost:1988`; nginx proxies `https://h.serj-tubin.com/api` to it.
- **Run locally**: `cd docker && make up` (Postgres, Redis, main service on 9000, file service, Adminer).
- **Testing**: Integration tests in `test/` need real dependencies (Docker). Set `ST_INT_TESTS=1` to run; suite uses dockertest for Postgres/Redis and starts the server.
- **Gymstats MCP**: MCP server at `/mcp` (Streamable HTTP) exposes gymstats schema and production data. Tools: `get_gymstats_context`, `get_exercises_for_time_range`, `get_exercise_types`, `get_exercise_history`, `get_exercise_percentages`, `get_avg_set_duration`. Code: `internal/gymstats/mcp/`; README there for Cursor setup. Auth: MCP secret or X-SERJ-TOKEN. Use when working on gymstats to get real DB schema and data.

## Main Frontend (serj-tubin-vue)

- **Stack**: Vue.js, Vuetify, Yarn.
- **Role**: Main site at www.serj-tubin.com (blog, netlog, notes, file box, visitor board, Spotify, weather, utils, gymstats Log/Stats/List).
- **API**: `VITE_API_ENDPOINT=https://h.serj-tubin.com/api` (dev and prod). File box: `VITE_FILE_BOX_ENDPOINT`, URL shortener endpoints, etc.
- **Run locally**: `yarn install && yarn serve-dev`; backend (e.g. Docker) as needed.

## Gymstats App (gymstats)

- **Stack**: React 18, Vite, Chart.js (react-chartjs-2), Axios, pnpm.
- **Role**: Single-user gym analytics (progress chart, progression rate, exercises table, muscle group + exercise type filters, metric toggles). Auth via backend `/a/login`; token in localStorage; `X-SERJ-TOKEN` header.
- **Backend API**: `/gymstats/stats/progress`, `/gymstats/stats/progression-rate`, `/gymstats/stats/exercises`, `/gymstats/types`, etc.
- **Run locally**: Backend via `serj-tubin-com/docker`; then `pnpm install && pnpm run dev`; `VITE_API_ENDPOINT=http://localhost:9000` in `.env`.
- **Docs**: In gymstats repo: `.agents/MISSION_AND_PROGRESS.md`, `SETUP.md`, `.agents/DB_EXPORT_IMPORT_PLAN.md`.
- **Next steps**: Add more features for gym exercises and progress insights. When deploying gymstats at **gymstats.serj-tubin.com**, add that origin to CORS in `internal/middleware/cors.go`.

## Deployment and Infrastructure

- **Host**: Hetzner (Ubuntu). **DNS**: Namecheap.
- **Sites**: www.serj-tubin.com, 2beens.online; API at **h.serj-tubin.com** (nginx path `/api`); file box at **file-box.serj-tubin.com**; gymstats (future) at **gymstats.serj-tubin.com**.
- **On server**: Nginx, main backend binary (systemd), file-box service (separate binary, same machine), PostgreSQL (single DB: `serj_blogs`), Redis. Also: rust-url-shortener, webhooks for redeploy; details can be added later.
- **CI/CD**: `.github/workflows/deploy.yml` — two jobs: **deploy-main-service** (mainservice) and **deploy-file-service** (filebox); both SSH to server and run remote redeploy script. Manual redeploy: `scripts/redeploy.sh` (builds main service + netlog backup; restarts serj-tubin-backend.service).

## Data and Clients

- **Netlog**: Browser extension (Chrome) installed on user's computers; sends visit data to backend.
- **Gymstats**: User uses only web clients (Vue section + React app). iOS app exists but is not in active use.

## Cross-Project Conventions

- **Auth**: Backend owns auth; Vue and gymstats use same login and `X-SERJ-TOKEN` header.
- **Gymstats**: Backend `/gymstats/*`; React app is main consumer; Vue has Log/Stats/List. When adding CORS for gymstats.serj-tubin.com, update `internal/middleware/cors.go`.
- **Local dev**: Backend first (Docker), then frontend(s) at `http://localhost:9000`. Follow Go and Vue best practices; no project-specific conventions beyond that.
