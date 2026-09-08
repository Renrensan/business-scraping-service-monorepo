export interface UserSession {
  adminId: string;
  username: string;
  email: string;
  createdAt: string;
  lastActiveAt: string;
}

export interface BusinessLead {
  id: string;
  job_id?: string;
  place_id: string;
  cid?: string;
  name: string;
  category: string;
  subcategories?: string[];
  address: string;
  street?: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
  latitude: number;
  longitude: number;
  rating: number;
  review_count: number;
  price_level?: string;
  claimed: boolean;
  business_status: string;
  website?: string;
  primary_phone?: string;
  emails?: string[];
  phones?: string[];
  facebook_url?: string;
  instagram_url?: string;
  linkedin_url?: string;
  twitter_url?: string;
  youtube_url?: string;
  tiktok_url?: string;
  google_maps_url?: string;
  enrichment_status: "NONE" | "PENDING" | "SUCCESS" | "FAILED";
  last_scraped_at: string;
  last_enriched_at?: string;
  created_at: string;
}

export interface ScrapeJob {
  id: string;
  status: "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED" | "CANCELLED";
  query: string;
  location: string;
  category?: string;
  max_results: number;
  enrich_contacts: boolean;
  total_scraped: number;
  total_enriched: number;
  error_count: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface OverviewMetrics {
  total_leads: number;
  enriched_leads: number;
  email_hit_rate: number;
  phone_hit_rate: number;
  active_jobs: number;
  completed_jobs: number;
  total_scraped_ever: number;
  next_scrape_at?: string;
  countdown_seconds?: number;
  auto_schedule_enabled?: boolean;
  schedule_interval?: string;
}

export interface ScheduleInfo {
  next_run_at: string;
  countdown_seconds: number;
  last_run_at?: string;
  interval_hours: number;
  auto_enabled: boolean;
  target_query: string;
  target_location: string;
}
