<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api/client';
  import type { OverviewMetrics, ScrapeJob, BusinessLead } from '$lib/types';
  import StatCard from '$lib/components/StatCard.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import NewJobModal from '$lib/components/NewJobModal.svelte';
  import { toast } from 'svelte-sonner';
  import { 
    Building2, Mail, Phone, Zap, Plus, 
    ArrowUpRight, RefreshCw, Layers, MapPin, Star,
    Clock, Play, ShieldCheck, Sparkles
  } from 'lucide-svelte';

  let metrics = $state<OverviewMetrics>({
    total_leads: 0,
    enriched_leads: 0,
    email_hit_rate: 0,
    phone_hit_rate: 0,
    active_jobs: 0,
    completed_jobs: 0,
    total_scraped_ever: 0,
    countdown_seconds: 0,
    auto_schedule_enabled: true,
    schedule_interval: 'Every 24h (Daily Batch)'
  });

  let recentJobs = $state<ScrapeJob[]>([]);
  let recentLeads = $state<BusinessLead[]>([]);
  let categoryStats = $state<{ category: string; count: number }[]>([]);
  let isLoading = $state(true);
  let isTriggering = $state(false);
  let showNewJobModal = $state(false);

  let countdownSeconds = $state(0);
  let timerInterval: any = null;

  function formatCountdown(sec: number): string {
    if (sec <= 0) return '00:00:00';
    const hours = Math.floor(sec / 3600);
    const minutes = Math.floor((sec % 3600) / 60);
    const seconds = sec % 60;
    return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
  }

  function formatNextRunTime(isoStr?: string): string {
    if (!isoStr) return '00:00:00 UTC (Daily Batch)';
    try {
      const d = new Date(isoStr);
      return d.toUTCString().replace('GMT', 'UTC');
    } catch {
      return isoStr;
    }
  }

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
        if (overviewData.countdown_seconds !== undefined) {
          countdownSeconds = overviewData.countdown_seconds;
        }
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

  async function triggerImmediateScrape() {
    try {
      isTriggering = true;
      await api.post('/api/v1/system/trigger-scrape', {
        query: 'Specialty Coffee',
        location: 'Seattle, WA',
        max_results: 50
      });
      toast.success('Immediate ingestion job triggered!');
      loadDashboardData();
    } catch (err: any) {
      toast.error(err.message || 'Failed to trigger ingestion');
    } finally {
      isTriggering = false;
    }
  }

  onMount(() => {
    loadDashboardData();

    // 1-second local countdown ticker
    timerInterval = setInterval(() => {
      if (countdownSeconds > 0) {
        countdownSeconds -= 1;
      }
    }, 1000);

    // 30-second data sync
    const syncInterval = setInterval(loadDashboardData, 30000);

    return () => {
      clearInterval(timerInterval);
      clearInterval(syncInterval);
    };
  });
</script>

<div class="space-y-6">
  <!-- Header & Actions -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-zinc-900">
    <div>
      <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Overview & Metrics</h1>
      <p class="text-xs font-mono text-zinc-500 mt-0.5">Real-time throughput, contact enrichment rates, and local database ingestion</p>
    </div>
    <div class="flex items-center gap-2.5">
      <button
        onclick={loadDashboardData}
        class="px-3.5 py-2 rounded-xl border border-zinc-800 bg-zinc-950 text-zinc-300 hover:bg-zinc-900 hover:text-white flex items-center gap-1.5 text-xs font-mono transition cursor-pointer"
      >
        <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Sync</span>
      </button>
      <button
        onclick={() => (showNewJobModal = true)}
        class="px-4 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black flex items-center gap-1.5 text-xs font-mono font-medium shadow-sm transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>New Extraction Job</span>
      </button>
    </div>
  </div>

  <!-- NEXT SCHEDULED SCRAPE LIVE TIMER BANNER -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-5 sm:p-6 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-6 relative overflow-hidden">
    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-mono bg-emerald-950/60 border border-emerald-800/80 text-emerald-400">
          <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
          Ingestion Cycle Active ({metrics.schedule_interval || 'Every 24h'})
        </span>
        <span class="text-[10px] font-mono text-zinc-500">• Smart Deduplication Enabled</span>
      </div>

      <div>
        <div class="text-xs font-mono text-zinc-400 uppercase tracking-wider">Next Scheduled Extraction Cycle</div>
        <div class="text-3xl sm:text-4xl font-mono font-bold text-white tracking-wider mt-1 flex items-baseline gap-3">
          <span>{formatCountdown(countdownSeconds)}</span>
          <span class="text-xs font-mono font-normal text-zinc-500 uppercase tracking-normal">remaining</span>
        </div>
        <p class="text-[11px] font-mono text-zinc-500 mt-1">
          Target Window: <span class="text-zinc-300">{formatNextRunTime(metrics.next_scrape_at)}</span>
        </p>
      </div>
    </div>

    <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 shrink-0">
      <button
        onclick={triggerImmediateScrape}
        disabled={isTriggering}
        class="px-4 py-2.5 rounded-xl border border-zinc-700 bg-zinc-900 hover:bg-zinc-800 text-white flex items-center justify-center gap-2 text-xs font-mono transition cursor-pointer disabled:opacity-50"
      >
        <Play class="w-3.5 h-3.5 fill-white text-white" />
        <span>{isTriggering ? 'Triggering...' : 'Trigger Now'}</span>
      </button>
      <button
        onclick={() => (showNewJobModal = true)}
        class="px-4 py-2.5 rounded-xl bg-white hover:bg-zinc-200 text-black flex items-center justify-center gap-1.5 text-xs font-mono font-medium shadow-sm transition cursor-pointer"
      >
        <Sparkles class="w-3.5 h-3.5 text-black" />
        <span>Manual Scrape</span>
      </button>
    </div>
  </div>

  <!-- KPI Grid -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <StatCard
      title="Total Leads"
      value={metrics.total_leads.toLocaleString()}
      subtitle="Local PostgreSQL storage"
      iconBg="bg-zinc-900 text-white"
    >
      {#snippet icon()}
        <Building2 class="w-4 h-4" />
      {/snippet}
    </StatCard>

    <StatCard
      title="Email Hit Rate"
      value={`${metrics.email_hit_rate.toFixed(1)}%`}
      subtitle={`${metrics.enriched_leads.toLocaleString()} verified profiles`}
      trend="Crawled"
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
      title="Active Jobs"
      value={metrics.active_jobs}
      subtitle={`${metrics.completed_jobs} completed batches`}
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
    <div class="lg:col-span-2 bg-zinc-950 border border-zinc-800/90 rounded-2xl p-5 shadow-sm">
      <div class="flex items-center justify-between pb-4 border-b border-zinc-800/80">
        <div>
          <h2 class="text-base font-display font-light text-white">Recently Ingested Leads</h2>
          <p class="text-[11px] font-mono text-zinc-500 mt-0.5">Fresh business listings saved to local database</p>
        </div>
        <a href="/businesses" class="text-xs font-mono text-zinc-300 hover:text-white flex items-center gap-1">
          <span>View All</span>
          <ArrowUpRight class="w-3.5 h-3.5" />
        </a>
      </div>

      <div class="mt-4 divide-y divide-zinc-900 text-xs font-mono">
        {#if recentLeads.length === 0}
          <div class="py-12 text-center text-zinc-500">
            No leads ingested yet. Launch your first extraction job!
          </div>
        {:else}
          {#each recentLeads as lead}
            <div class="py-3 flex items-center justify-between gap-3 hover:bg-zinc-900/40 px-2.5 rounded-xl transition">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-white truncate">{lead.name}</span>
                  <Badge variant="default">{lead.category}</Badge>
                </div>
                <div class="flex items-center gap-3 mt-1 text-[11px] text-zinc-500">
                  <span class="flex items-center gap-1 truncate"><MapPin class="w-3 h-3 text-zinc-600" /> {lead.city || lead.address}</span>
                  {#if lead.emails && lead.emails.length > 0}
                    <span class="text-emerald-400">✉️ {lead.emails.length} email</span>
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

    <!-- Category Breakdown & Batch Jobs -->
    <div class="space-y-6">
      <!-- Category Distribution -->
      <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-5 shadow-sm">
        <h2 class="text-base font-display font-light text-white mb-3 flex items-center gap-2">
          <Layers class="w-4 h-4 text-zinc-400" />
          <span>Top Categories Scraped</span>
        </h2>
        <div class="space-y-2 text-xs font-mono">
          {#if categoryStats.length === 0}
            <div class="py-6 text-center text-zinc-500 text-[11px]">No category data yet</div>
          {:else}
            {#each categoryStats as cat}
              <div class="flex items-center justify-between p-2.5 rounded-xl bg-black border border-zinc-800">
                <span class="text-zinc-300 font-medium truncate">{cat.category}</span>
                <span class="text-white font-mono font-bold bg-zinc-900 px-2 py-0.5 rounded text-[11px] border border-zinc-700">{cat.count}</span>
              </div>
            {/each}
          {/if}
        </div>
      </div>

      <!-- Recent Jobs Quick Status -->
      <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-5 shadow-sm">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-base font-display font-light text-white flex items-center gap-2">
            <Zap class="w-4 h-4 text-zinc-400" />
            <span>Recent Jobs</span>
          </h2>
          <a href="/jobs" class="text-[11px] font-mono text-zinc-400 hover:text-white">Manage</a>
        </div>
        <div class="space-y-2 text-xs font-mono">
          {#if recentJobs.length === 0}
            <div class="py-6 text-center text-zinc-500 text-[11px]">No jobs executed yet</div>
          {:else}
            {#each recentJobs as job}
              <div class="p-2.5 rounded-xl bg-black border border-zinc-800 flex items-center justify-between">
                <div class="min-w-0">
                  <div class="font-medium text-white truncate">{job.query}</div>
                  <div class="text-[10px] text-zinc-500">{job.location} • {job.total_scraped} leads</div>
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