<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { OverviewMetrics, ScrapeJob, BusinessLead } from '$lib/types';
  import StatCard from '$lib/components/StatCard.svelte';
  import ScrapeHealthCard from '$lib/components/ScrapeHealthCard.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import { 
    Building2, Mail, Phone, Zap, 
    ArrowUpRight, RefreshCw, Layers, MapPin, Star
  } from 'lucide-svelte';

  let metrics = $state<OverviewMetrics>({
    total_leads: 0,
    enriched_leads: 0,
    leads_with_email: 0,
    leads_with_phone: 0,
    email_hit_rate: 0,
    phone_hit_rate: 0,
    active_jobs: 0,
    completed_jobs: 0,
    total_scraped_ever: 0
  });

  let recentJobs = $state.raw<ScrapeJob[]>([]);
  let recentLeads = $state.raw<BusinessLead[]>([]);
  let categoryStats = $state.raw<{ category: string; count: number }[]>([]);
  let isLoading = $state(true);

  async function loadDashboardData() {
    try {
      isLoading = true;
      const [overviewData, jobsData, leadsData, catData] = await Promise.all([
        api.get<OverviewMetrics>('/api/v1/metrics/overview'),
        api.get<{ items: ScrapeJob[] }>('/api/v1/jobs', { page: 1, page_size: 5 }),
        api.get<{ items: BusinessLead[] }>('/api/v1/businesses', { page: 1, page_size: 6 }),
        api.get<{ category: string; count: number }[]>('/api/v1/metrics/categories', { limit: 6 })
      ]);

      if (overviewData) {
        metrics = overviewData;
      }
      if (jobsData?.items) recentJobs = jobsData.items;
      if (leadsData?.items) recentLeads = leadsData.items;
      if (Array.isArray(catData)) categoryStats = catData;
    } catch (e) {
      console.error('Failed to load overview data', e);
    } finally {
      isLoading = false;
    }
  }

  onMount(() => {
    loadDashboardData();
    const syncInterval = setInterval(loadDashboardData, 30000);
    return () => clearInterval(syncInterval);
  });
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-zinc-900">
    <div>
      <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Overview</h1>
      <p class="text-xs text-zinc-500 mt-0.5">Google Maps business scraper with automated scheduling and contact enrichment</p>
    </div>
    <div class="flex items-center gap-2.5">
      <button
        onclick={loadDashboardData}
        class="px-3.5 py-2 rounded-xl border border-zinc-800 bg-zinc-950 text-zinc-300 hover:bg-zinc-900 hover:text-white flex items-center gap-1.5 text-xs font-medium transition cursor-pointer"
      >
        <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Sync</span>
      </button>
    </div>
  </div>

  <ScrapeHealthCard />

  <!-- KPI Grid -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <StatCard
      title="Total Leads"
      value={metrics.total_leads.toLocaleString()}
      subtitle="Saved in database"
      iconBg="bg-zinc-900 text-white"
    >
      {#snippet icon()}
        <Building2 class="w-4 h-4" />
      {/snippet}
    </StatCard>

    <StatCard
      title="Emails Found"
      value={(metrics.leads_with_email ?? 0).toLocaleString()}
      subtitle={`${metrics.email_hit_rate.toFixed(1)}% hit rate`}
      iconBg="bg-zinc-900 text-white"
    >
      {#snippet icon()}
        <Mail class="w-4 h-4" />
      {/snippet}
    </StatCard>

    <StatCard
      title="Phone Reach"
      value={`${metrics.phone_hit_rate.toFixed(1)}%`}
      subtitle="Direct business lines"
      iconBg="bg-zinc-900 text-white"
    >
      {#snippet icon()}
        <Phone class="w-4 h-4" />
      {/snippet}
    </StatCard>

    <StatCard
      title="Scrape Jobs"
      value={metrics.completed_jobs}
      subtitle={`${metrics.active_jobs} running now`}
      iconBg="bg-zinc-900 text-white"
    >
      {#snippet icon()}
        <Zap class="w-4 h-4" />
      {/snippet}
    </StatCard>
  </div>

  <!-- Main Content Split -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Recent Leads -->
    <div class="lg:col-span-2 bg-zinc-950 border border-zinc-800 rounded-2xl p-5 shadow-sm">
      <div class="flex items-center justify-between pb-4 border-b border-zinc-800">
        <div>
          <h2 class="text-base font-semibold text-white">Recently Scraped Leads</h2>
          <p class="text-xs text-zinc-500 mt-0.5">Latest business listings added to database</p>
        </div>
        <a href="/businesses" class="text-xs font-medium text-zinc-300 hover:text-white flex items-center gap-1">
          <span>View All</span>
          <ArrowUpRight class="w-3.5 h-3.5" />
        </a>
      </div>

      <div class="mt-4 divide-y divide-zinc-900 text-xs">
        {#if recentLeads.length === 0}
          <div class="py-12 text-center text-zinc-500">
            No business listings scraped yet. Start a scrape from the Scrape Now page to populate leads.
          </div>
        {:else}
          {#each recentLeads as lead (lead.id)}
            <div class="py-3 flex items-center justify-between gap-3 hover:bg-zinc-900/40 px-2.5 rounded-xl transition">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-white truncate">{lead.name}</span>
                  <Badge variant="default">{lead.category}</Badge>
                </div>
                <div class="flex items-center gap-3 mt-1 text-xs text-zinc-500">
                  <span class="flex items-center gap-1 truncate"><MapPin class="w-3 h-3 text-zinc-600" /> {lead.city || lead.address}</span>
                  {#if lead.emails && lead.emails.length > 0}
                    <span class="text-emerald-400 font-medium">✉️ {lead.emails.length} email</span>
                  {/if}
                </div>
              </div>
              <div class="flex items-center gap-1.5 shrink-0 text-white font-mono text-xs">
                <Star class="w-3.5 h-3.5 fill-white text-white" />
                <span>{lead.rating.toFixed(1)}</span>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Category Breakdown & Recent Jobs -->
    <div class="space-y-6">
      <!-- Category Distribution -->
      <div class="bg-zinc-950 border border-zinc-800 rounded-2xl p-5 shadow-sm">
        <h2 class="text-base font-semibold text-white mb-3 flex items-center gap-2">
          <Layers class="w-4 h-4 text-zinc-400" />
          <span>Top Categories</span>
        </h2>
        <div class="space-y-2 text-xs">
          {#if categoryStats.length === 0}
            <div class="py-6 text-center text-zinc-500">No categories found yet</div>
          {:else}
            {#each categoryStats as cat (cat.category)}
              <div class="flex items-center justify-between p-2.5 rounded-xl bg-black border border-zinc-800">
                <span class="text-zinc-300 font-medium truncate">{cat.category}</span>
                <span class="text-white font-mono font-bold bg-zinc-900 px-2 py-0.5 rounded text-xs border border-zinc-700">{cat.count}</span>
              </div>
            {/each}
          {/if}
        </div>
      </div>

      <!-- Recent Scrape Jobs -->
      <div class="bg-zinc-950 border border-zinc-800 rounded-2xl p-5 shadow-sm">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-base font-semibold text-white flex items-center gap-2">
            <Zap class="w-4 h-4 text-zinc-400" />
            <span>Recent Scrapes</span>
          </h2>
          <a href="/scraper" class="text-xs text-zinc-400 hover:text-white font-medium">Manage →</a>
        </div>
        <div class="space-y-2 text-xs">
          {#if recentJobs.length === 0}
            <div class="py-6 text-center text-zinc-500">No scrape jobs executed yet</div>
          {:else}
            {#each recentJobs as job (job.id)}
              <div class="p-2.5 rounded-xl bg-black border border-zinc-800 flex items-center justify-between">
                <div class="min-w-0">
                  <div class="font-medium text-white truncate">{job.query}</div>
                  <div class="text-xs text-zinc-500 mt-0.5">{job.location} • {job.total_scraped} leads</div>
                </div>
                <Badge variant={job.status === 'COMPLETED' ? 'success' : job.status === 'PROCESSING' ? 'warning' : 'default'}>
                  {job.status}
                </Badge>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
