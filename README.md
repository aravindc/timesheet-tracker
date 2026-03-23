# Timesheet Tracker

A self-hosted time tracking application with project management, hourly rate tracking, and UK payslip generation.

**Live:** https://timesheet.pravitha.in

---

## Stack

| Layer | Technology |
|---|---|
| Frontend | React 18 + TypeScript + Vite + TailwindCSS |
| Backend | Go (Gin) |
| Database | PostgreSQL (Supabase) |
| Auth | JWT (Bearer token) + bcrypt |
| Proxy | Nginx |
| Deployment | Docker Compose |

---

## Features

- **Time Tracking** — Log work days with start/end times and break hours; total hours are calculated automatically
- **Project Management** — Create projects and assign work entries to them
- **Calendar View** — Visual month/year calendar showing all entries
- **Statistics** — Total hours and entry count per project
- **Hourly Rates** — Track rate history with effective dates; mark one rate as current
- **Payslip Generation** — Monthly payslips with UK Income Tax (2024/25) and National Insurance breakdowns
- **IP Whitelisting** — Registration endpoint restricted by IP to prevent public sign-ups
- **Dark Mode** — Persisted in localStorage

---

## Project Structure

```
timesheet-tracker/
├── backend/
│   ├── main.go          # Entry point: DB connect, JWT init, server start
│   ├── server.go        # Router setup, DB schema init, CORS
│   ├── models.go        # All struct definitions
│   ├── auth.go          # register, login, verify, generateToken
│   ├── middleware.go    # JWT auth, IP whitelist, env helpers
│   ├── projects.go      # Project CRUD + stats handler
│   ├── workdays.go      # Work day CRUD
│   ├── rates.go         # Hourly rate CRUD
│   ├── payslip.go       # Payslip handler + UK tax/NI calculations
│   ├── logger.go        # Zap logger (dev/prod)
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── App.tsx          # Main app: auth, routing, state
│   │   ├── CalendarView.tsx # Calendar interface
│   │   ├── EntriesList.tsx  # Work day list
│   │   ├── TimerForm.tsx    # Time entry form
│   │   ├── ProjectStats.tsx # Statistics view
│   │   ├── api.ts           # API client
│   │   ├── types.ts         # TypeScript types
│   │   └── utils.ts         # Utility functions
│   ├── nginx.conf       # SPA routing (try_files fallback)
│   └── Dockerfile
├── proxy/
│   └── nginx.conf       # Routes /api/* → backend, / → frontend
├── docker-compose.yml
└── .env.app             # Environment variables (not committed)
```

---

## API Routes

### Public
| Method | Path | Description |
|---|---|---|
| `POST` | `/api/auth/register` | Register (IP whitelisted) |
| `POST` | `/api/auth/login` | Login, returns JWT |
| `GET` | `/api/auth/verify` | Verify token validity |
| `GET` | `/api/healthz/live` | Liveness probe |

### Protected (Bearer token required)
| Method | Path | Description |
|---|---|---|
| `GET` | `/api/projects` | List projects |
| `POST` | `/api/projects` | Create project |
| `PUT` | `/api/projects/:id` | Update project |
| `DELETE` | `/api/projects/:id` | Delete project |
| `GET` | `/api/stats` | Project stats (hours, entry count) |
| `GET` | `/api/workdays` | List work days (last 100) |
| `GET` | `/api/workdays/:year/:month` | Work days for a month (`?project_id=` optional) |
| `POST` | `/api/workdays` | Create work day |
| `PUT` | `/api/workdays/:id` | Update work day |
| `DELETE` | `/api/workdays/:id` | Delete work day |
| `GET` | `/api/hourly-rates` | List user's hourly rates |
| `POST` | `/api/hourly-rates` | Create rate (marks others as not current) |
| `PUT` | `/api/hourly-rates/:id` | Update rate |
| `DELETE` | `/api/hourly-rates/:id` | Delete rate |
| `GET` | `/api/payslip/:month` | Generate payslip (format: `Jan-2026`) |

---

## Database Schema

```sql
users         (id, username, password_hash, created_at)
projects      (id, name, description, created_at)
work_days     (id, date, project_id, project_name, start_time, end_time,
               break_hours, total_hours, created_at, updated_at)
hourly_rates  (id, user_id, rate, is_current, effective_from, created_at)
```

Schema is auto-created on startup via `initDB()` in `server.go`.

---

## Running Locally

### With Docker Compose

```bash
cp .env.app.example .env.app   # fill in your values
docker compose up --build
```

App will be available at `http://localhost:8089`.

### Backend Only

```bash
cd backend
export JWT_SECRET=<secret>
export DATABASE_URL=<postgres-connection-string>
export WHITELIST_IPS=127.0.0.1,::1
go run .
```

### Frontend Only

```bash
cd frontend
npm install
npm run dev
```

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `JWT_SECRET` | Yes | JWT signing secret |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `GIN_MODE` | No | `debug` or `release` (default: debug) |
| `APP_ENV` | No | `development` or `production` (affects log format) |
| `WHITELIST_IPS` | No | Comma-separated IPs allowed to register. Defaults to `127.0.0.1,::1` |
| `WHITELIST_CIDRS` | No | Comma-separated CIDR blocks allowed to register |
| `TRUSTED_PROXIES` | No | Comma-separated proxy IPs. Defaults to `127.0.0.1,::1` |

---

## Agent Prompt

> Use this section to give an AI agent a quick understanding of the project before making changes.

This is a self-hosted timesheet tracking web app. It is a monorepo with two main parts:

**Backend** (`/backend`) is a Go REST API using the Gin framework. It connects to a PostgreSQL database (Supabase) and uses JWT (Bearer token) for auth. The code is split across focused files: `main.go` (entry point), `server.go` (router + DB init), `models.go` (all structs), `auth.go`, `middleware.go`, `projects.go`, `workdays.go`, `rates.go`, `payslip.go`, and `logger.go`. All files are in `package main`. The server runs on port 8080. Registration is IP-whitelisted. Passwords are bcrypt-hashed. The payslip endpoint calculates UK Income Tax and National Insurance for a given month using the user's current hourly rate and their logged work hours for that month.

**Frontend** (`/frontend`) is a React + TypeScript SPA built with Vite and styled with TailwindCSS. It is a single large `App.tsx` that manages auth state, view routing (no React Router — view state is local), and API calls. Supporting components include `CalendarView.tsx`, `TimerForm.tsx`, `EntriesList.tsx`, and `ProjectStats.tsx`. The API client is in `api.ts`. Auth tokens are stored in localStorage.

**Deployment** uses Docker Compose with three services: `backend` (Go binary), `frontend` (React served via Nginx), and `proxy` (Nginx reverse proxy on port 8089 that routes `/api/*` → backend and `/` → frontend). Environment variables for the backend are loaded from `.env.app`.

**Key conventions:**
- All backend types are defined in `models.go`
- `total_hours` is always computed server-side from `(end_time - start_time) - break_hours`
- The `is_current` flag on `hourly_rates` is managed transactionally — creating a new rate atomically clears the old one
- The payslip endpoint uses monthly gross pay (not annual) against prorated UK 2024/25 tax bands
- The DB schema is created on startup if tables do not exist — no migration tooling
