export interface UserSession {
  adminId: string;
  username: string;
  email: string;
  createdAt: string;
  lastActiveAt: string;
}

/** Admin-assigned temperature of a contacted lead. Not-contacted leads have no status. */
export type LeadStatus = 'cold' | 'warm' | 'hot';

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
  contacted?: boolean;
  contacted_at?: string;
  lead_status?: LeadStatus | null;
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
  leads_with_email?: number;
  leads_with_phone?: number;
  email_hit_rate: number;
  phone_hit_rate: number;
  active_jobs: number;
  completed_jobs: number;
  total_scraped_ever: number;
  next_scrape_at?: string;
  countdown_seconds?: number;
  auto_schedule_enabled?: boolean;
  schedule_interval?: string;
  only_new_businesses?: boolean;
  require_website?: boolean;
  website_filter?: 'any' | 'require_website' | 'no_website';
  recency_filter?: 'any' | 'only_new' | 'established';
  target_locations?: string[];
  target_queries?: string[];
}

export interface ScheduleInfo {
  next_run_at: string;
  countdown_seconds: number;
  last_run_at?: string;
  interval_hours: number;
  auto_enabled: boolean;
  target_query: string;
  target_location: string;
  target_category?: string;
  target_queries?: string[];
  target_locations?: string[];
  target_categories?: string[];
  only_new_businesses?: boolean;
  require_website?: boolean;
  website_filter?: 'any' | 'require_website' | 'no_website';
  recency_filter?: 'any' | 'only_new' | 'established';
  max_review_count_for_new?: number;
  max_pages_per_run?: number;
  page_size?: number;
  max_results?: number;
  enrich_contacts?: boolean;
  engine?: string;
}

export interface ScraperKeyword {
  id: string;
  keyword: string;
  next_page_token: string;
  current_page: number;
  total_leads_scraped: number;
  last_scraped_at?: string;
  status: 'ready' | 'processing' | 'paginating' | 'completed';
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListKeywordResponse {
  items: ScraperKeyword[];
  total_count: number;
  total_pages: number;
  page: number;
  page_size: number;
}

export interface KeywordScrapeResult {
  job_id?: string;
  keyword: string;
  leads_found?: number;
  pages_traversed?: number;
  next_page_token?: string;
  status: 'ready' | 'processing' | 'paginating' | 'completed';
  cycle_completed?: boolean;
  current_page?: number;
  total_leads_scraped?: number;
  message?: string;
}
