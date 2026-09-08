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
