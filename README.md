# High-Speed Google Maps Extraction & Lead Enrichment Service

A high-performance, self-hosted data extraction and lead generation platform designed to pull thousands of Google Maps business profiles into structured datasets and spreadsheets within minutes.

## Features
- ⚡ **Dual Extraction Engine**:
  - **Mode A (High-Speed Web Scraper - Default)**: Zero API key cost, no 60-result pagination cap, smart proxy rotation.
  - **Mode B (Official Google Places API New)**: Optimized field masking (`X-Goog-FieldMask`) to maximize free tier quotas (10,000 Essentials / 5,000 Pro calls).
- 🔍 **Deep Contact Enrichment (PhantomBuster Style)**: Automated website crawling (`/`, `/contact`, `/about`) to extract direct emails, phone numbers, and social media channels (LinkedIn, Instagram, Facebook, X, YouTube, TikTok).
- 🚀 **Golang Backend**: High-concurrency worker pool, Chi HTTP router, PostgreSQL persistence, streamable CSV/Excel export.
- 🎨 **Svelte 5 Dashboard**: Ultra-lightweight reactive UI with Tailwind CSS + bits-ui / shadcn, real-time job progress tracking, multi-criteria filtering, and analytics.
- 🔐 **Redis Single-User Auth**: Secure session store with HttpOnly cookies, following the security architecture of `paradapp/paradapp-admin-dashboard`.

## Documentation
- [Service Architecture](docs/SERVICE_ARCHITECTURE.md)
- [Google Places API vs Scraping Analysis](docs/GOOGLE_API_AND_SCRAPING_ANALYSIS.md)
- [Google Places API (New) Technical Spec](docs/GOOGLE_PLACES_API_SPEC.md)

## Development Quickstart (with Mise)
```bash
# 1. Start local storage (Postgres 16 + Redis)
mise run db-run-dev

# 2. Run backend & frontend concurrently
mise run dev
```

## Operations

**Keywords.** "By area" keywords (`villa · Canggu`) search a term inside an area's rectangle with an
optional Google Places type. Google returns at most 60 places per search, so an area that hits 60 is
split into four smaller cells (up to 3 levels) and the cycle continues over later runs. "Plain text"
keywords are sent to Google as typed and stop at 60. Keywords whose runs find nothing new are scheduled
less often (1, 2, 4 … 30 days); the Scrape button ignores that.

**Jobs survive restarts.** Every scrape is a row in `scrape_jobs` claimed by a worker. On deploy,
running jobs get `SHUTDOWN_GRACE_SECONDS` to finish and are otherwise queued again; jobs of a process
that died are found by their stale heartbeat and retried (up to 3 attempts).

**Quota and budget.** When Google's daily quota or the daily request limit (Settings) is reached,
scraping pauses until Google's reset (midnight Pacific time) and queued jobs are cancelled. Saving a
new API key or changing the limit lifts the pause. The dashboard's *Scraper health* card shows the
queue, failures, quota and schedule.

**Alerts.** Set `EVOLUTION_API_URL`, `EVOLUTION_API_KEY`, `EVOLUTION_INSTANCE` and
`ALERT_WHATSAPP_NUMBER` to get WhatsApp messages when the quota is reached, a job fails or a keyword
returns no results (each at most once an hour). Without them, alerts are only logged.

**Database migrations** live in `backend/internal/repository/migrations/` and run on startup, once
each, recorded in `schema_migrations`. To change the schema, add a file with the next number; never
edit one that has been deployed.

**Tests.** `mise run test` (image builds run it first). Database tests run when `TEST_DATABASE_URL`
points to a disposable Postgres; CI runs them on every push.
