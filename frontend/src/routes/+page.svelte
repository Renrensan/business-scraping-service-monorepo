<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { OverviewMetrics, ScrapeJob, BusinessLead } from '$lib/types';
  import StatCard from '$lib/components/StatCard.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import NewJobModal from '$lib/components/NewJobModal.svelte';
  import { 
    Building2, Mail, Phone, Zap, Plus, 
    ArrowUpRight, RefreshCw, Layers, MapPin, Star
  } from 'lucide-svelte';

  let metrics = $state<OverviewMetrics>({
    total_leads: 0,
    enriched_leads: 0,
    email_hit_rate: 0,
    phone_hit_rate: 0,
    active_jobs: 0,
    completed_jobs: 0,
    total_scraped_ever: 0
  });

  let recentJobs = $state<ScrapeJob[]>([]);
  let recentLeads = $state<BusinessLead[]>([]);
  let categoryStats = $state<{ category: string; count: number }[]>([]);
  let isLoading = $state(true);
  let showNewJobModal = $state(false);

  async function loadDashboardData() {
    try {
      isLoading = true;
      const [overviewData, jobsData, leadsData, catData] = await Promise.all([
        api.get<OverviewMetrics>('/api/v1/metrics/overview'),
        api.get<{ items: ScrapeJob[] }>('/api/v1/jobs', { page: 1, page_size: 5 }),
        api.get<{ items: BusinessLead[] }>('/api/v1/businesses', { page: 1, page_size: 6 }),
        api.get<{ category: string; count: number }[]>('/api/v1/metrics/categories', { limit: 6 })
      ]);

      if (overviewData) metrics = overviewData;
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
  });
</script>

<div class="space-y-6">
  <!-- Header & Actions -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h1 class="text-xl font-bold text-white tracking-tight">Overview & Analytics</h1>
      <p class="text-xs text-slate-400 mt-1">Real-time throughput, lead enrichment rates, and batch ingestion status</p>
    </div>
    <div class="flex items-center gap-2.5">
      <button
        onclick={loadDashboardData}
        class="px-3.5 py-2 rounded-xl border border-slate-700 bg-slate-800 text-slate-300 hover:bg-slate-700 flex items-center gap-1.5 text-xs font-medium transition cursor-pointer"
      >
        <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </button>
      <button
        onclick={() => (showNewJobModal = true)}
        class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white flex items-center gap-1.5 text-xs font-medium shadow-md shadow-indigo-600/20 transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>New Scrape Job</span>
      </button>
    </div>
  </div>

  <!-- KPI Grid -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <StatCard
      title="Total Leads"
      value={metrics.total_leads.toLocaleString()}
      subtitle="In local PostgreSQL lead lake"
      iconBg="bg-indigo-500/10 text-indigo-400"
    >
      {#snippet icon()}
        <Building2 class="w-5 h-5" />
      {/snippet}
    </StatCard>

    <StatCard
      title="Discovered Emails"
      value={`${metrics.email_hit_rate.toFixed(1)}%`}
      subtitle={`${metrics.enriched_leads.toLocaleString()} profiles enriched`}
      trend="Active"
      iconBg="bg-amber-500/10 text-amber-400"
    >
      {#snippet icon()}
        <Mail class="w-5 h-5" />
      {/snippet}
    </StatCard>

    <StatCard
      title="Phone Reach"
      value={`${metrics.phone_hit_rate.toFixed(1)}%`}
      subtitle="Direct business numbers"
      iconBg="bg-emerald-500/10 text-emerald-400"
    >
      {#snippet icon()}
        <Phone class="w-5 h-5" />
      {/snippet}
    </StatCard>

    <StatCard
      title="Active Ingestion Jobs"
      value={metrics.active_jobs}
      subtitle={`${metrics.completed_jobs} completed`}
      iconBg="bg-violet-500/10 text-violet-400"
    >
      {#snippet icon()}
        <Zap class="w-5 h-5" />
      {/snippet}
    </StatCard>
  </div>

  <!-- Main Content Split -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Recent Leads -->
    <div class="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-sm">
      <div class="flex items-center justify-between pb-4 border-b border-slate-800">
        <div>
          <h2 class="text-sm font-bold text-white">Recently Ingested Leads</h2>
          <p class="text-[11px] text-slate-400 mt-0.5">Fresh business listings saved to database</p>
        </div>
        <a href="/businesses" class="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1 font-medium">
          <span>View All</span>
          <ArrowUpRight class="w-3.5 h-3.5" />
        </a>
      </div>

      <div class="mt-4 divide-y divide-slate-800/60 text-xs">
        {#if recentLeads.length === 0}
          <div class="py-12 text-center text-slate-400">
            No leads ingested yet. Start your first scrape job!
          </div>
        {:else}
          {#each recentLeads as lead}
            <div class="py-3 flex items-center justify-between gap-3 hover:bg-slate-800/30 px-2 rounded-xl transition">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-semibold text-slate-200 truncate">{lead.name}</span>
                  <Badge variant="purple">{lead.category}</Badge>
                </div>
                <div class="flex items-center gap-3 mt-1 text-[11px] text-slate-400">
                  <span class="flex items-center gap-1 truncate"><MapPin class="w-3 h-3 text-slate-400" /> {lead.city || lead.address}</span>
                  {#if lead.emails && lead.emails.length > 0}
                    <span class="text-emerald-400 font-mono">✉️ {lead.emails.length} email</span>
                  {/if}
                </div>
              </div>
              <div class="flex items-center gap-1.5 shrink-0 text-amber-400 font-medium">
                <Star class="w-3.5 h-3.5 fill-amber-400" />
                <span>{lead.rating.toFixed(1)}</span>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Category Breakdown & Batch Jobs -->
    <div class="space-y-6">
      <!-- Category Distribution -->
      <div class="bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-sm">
        <h2 class="text-sm font-bold text-white mb-3 flex items-center gap-2">
          <Layers class="w-4 h-4 text-indigo-400" />
          <span>Top Categories Scraped</span>
        </h2>
        <div class="space-y-2.5 text-xs">
          {#if categoryStats.length === 0}
            <div class="py-6 text-center text-slate-400 text-[11px]">No category data yet</div>
          {:else}
            {#each categoryStats as cat}
              <div class="flex items-center justify-between p-2 rounded-lg bg-slate-950/50 border border-slate-800/80">
                <span class="text-slate-300 font-medium truncate">{cat.category}</span>
                <span class="text-indigo-400 font-mono font-bold bg-indigo-500/10 px-2 py-0.5 rounded text-[11px] border border-indigo-500/20">{cat.count}</span>
              </div>
            {/each}
          {/if}
        </div>
      </div>

      <!-- Recent Jobs Quick Status -->
      <div class="bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-sm">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-sm font-bold text-white flex items-center gap-2">
            <Zap class="w-4 h-4 text-amber-400" />
            <span>Recent Jobs</span>
          </h2>
          <a href="/jobs" class="text-[11px] text-indigo-400 hover:underline">Manage</a>
        </div>
        <div class="space-y-2 text-xs">
          {#if recentJobs.length === 0}
            <div class="py-6 text-center text-slate-400 text-[11px]">No jobs executed yet</div>
          {:else}
            {#each recentJobs as job}
              <div class="p-2.5 rounded-xl bg-slate-950/50 border border-slate-800 flex items-center justify-between">
                <div class="min-w-0">
                  <div class="font-medium text-slate-200 truncate">{job.query}</div>
                  <div class="text-[10px] text-slate-400">{job.location} • {job.total_scraped} leads</div>
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

<NewJobModal
  isOpen={showNewJobModal}
  onClose={() => (showNewJobModal = false)}
  onJobCreated={loadDashboardData}
/>