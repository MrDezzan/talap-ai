<h1 align="center">Talap.ai</h1>
<p align="center"><b>Build your career path in 5 minutes.</b><br>An AI platform that helps students find a profession, match with grants and follow a step-by-step roadmap.</p>

<p align="center">
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/Vite-646CFF?style=flat-square&logo=vite&logoColor=white" alt="Vite">
  <img src="https://img.shields.io/badge/Go-Gin-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/OpenAI-API-412991?style=flat-square&logo=openai&logoColor=white" alt="OpenAI">
  <img src="https://img.shields.io/badge/JWT-auth-000000?style=flat-square&logo=jsonwebtokens&logoColor=white" alt="JWT">
</p>

<p align="center"><a href="https://canva.link/talapaiproject">Project presentation</a> · Built in Astana, Kazakhstan</p>

---

## The problem

Students in Kazakhstan often pick a profession and a university without a clear picture of the options, and they miss grants because the information is scattered across dozens of sites. Talap.ai puts the options, the grants and a concrete plan in one place.

## Features

**Personal AI assistant.** Reads the user's profile (skills, achievements, interests) and builds a personal development strategy. A match score shows how well the user fits a given grant or career.

**Dynamic roadmap.** A step-by-step plan toward the chosen goal. Every step has a description, an estimated duration and useful resources, and the plan adapts to what the user already knows.

**AI consultant.** A chat for career questions. It can draft a CV from the conversation and the portfolio, and suggest courses or an exam-prep plan.

**Smart portfolio.** One place for skills, exam results (UNT/ENT) and awards, with CV export to PDF.

**Career and grant catalogs.** Filter by category, country and amount, track grant deadlines and see salary and demand trends for professions.

**Live search.** Opens with `Cmd + K` and searches professions and grants at once, with an instant preview.

**Three languages.** Russian, Kazakh and English across the whole interface.

**Admin tools.** An admin dashboard for managing grants and professions, plus a background importer that pulls grants from external sources and queues them for manual approval.

## Tech stack

| Layer | Technology |
|---|---|
| Frontend | React 18, Vite, Context API (auth, state, localization), custom CSS, Lucide icons, jsPDF + html2canvas for CV export |
| Backend | Go, Gin, GORM, JWT, Google OAuth |
| AI | OpenAI-compatible API (OpenAI or OpenRouter) for chat, roadmaps and CV drafting |
| Quality | Handler and middleware tests, rate limiting on auth routes, structured logging, load test script |
| Deploy | Dockerfile for the API, `vercel.json` for the frontend |

## Architecture

```
React SPA (Vite)  ──HTTPS/JSON──▶  Go API (Gin)  ──▶  Database (GORM)
                                       │
                                       ├──▶ OpenAI-compatible LLM
                                       ├──▶ Google OAuth
                                       └──▶ Grant importer (background worker)
```

```
backend/
  main.go         routes and server setup
  handlers/       auth, grants, AI, admin
  middleware/     JWT auth, rate limiting
  importer/       background grant importer
  config/         database and env setup
  monitoring/     webhook alerts
src/
  screens/        Dashboard, Catalog, Grants, Roadmap, Portfolio, Chat, Quiz, Profile, Admin
  pages/          Landing, Login, Register, Forgot password
  context/        auth, language and chat state
```

## Run locally

**Backend**

```bash
cd backend
cp .env.example .env      # add your LLM key, Google OAuth credentials and database settings
go run main.go            # http://localhost:8080
```

**Frontend**

```bash
npm install
npm run dev               # http://localhost:5173
```

The frontend talks to `http://localhost:8080/api` on localhost. In production set `VITE_API_URL`.

More backend details are in [backend/README.md](backend/README.md).

## My role

I built the whole product alone: the Go backend and its API, the OpenAI integration behind the consultant and roadmap generation, the grant importer, and the React frontend with three-language support.
