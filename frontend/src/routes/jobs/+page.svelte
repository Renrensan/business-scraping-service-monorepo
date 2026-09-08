<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ScrapeJob } from '$lib/types';
  import Badge from '$lib/components/Badge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import NewJobModal from '$lib/components/NewJobModal.svelte';
  import { 
    Zap, Plus, RefreshCw, CheckCircle2, 
    AlertCircle, Clock, Search, MapPin 
  } from 'lucide-svelte';

  let jobs = $state<ScrapeJob[]>([]);
  let totalCount = $state(0);
  let totalPages = $state(1);
  let page = $state(1);
  let pageSize = $state(15);
  let isLoading = $state(true);
  let showNewJobModal = $state(false);

  async function fetchJobs() {
    try {
      isLoading = true;
      const res = await api.get<{
        items: ScrapeJob[];
        total_count: number;
        total_pages: number;
      }>('/api/v1/jobs', { page, page_size: pageSize });

      if (res) {
        jobs = res.items || [];
        totalCount = res.total_count || 0;
        totalPages = res.total_pages || 1;
      }
    } catch (e) {
      console.error('Failed to fetch jobs', e);
    } finally {
      isLoading = false;
    }
  }

  onMount(() => {
    fetchJobs();
    const interval = setInterval(fetchJobs, 5000); // auto-refresh active jobs
    return () => clearInterval(interval);
  });
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h1 class="text-xl font-bold text-white tracking-tight">Scrape Jobs & Ingestion Pipeline</h1>
      <p class="text-xs text-slate-400 mt-1">Monitor background extraction workers, daily batches, and contact enrichments</p>
    </div>
    <div class="flex items-center gap-2.5">
      <button
        onclick={fetchJobs}
        class="px-3.5 py-2 rounded-xl border border-slate-700 bg-slate-800 text-slate-300 hover:bg-slate-700 flex items-center gap-1.5 text-xs font-medium transition cursor-pointer"
      >
        <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh Status</span>
      </button>
      <button
        onclick={() => (showNewJobModal = true)}
        class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white flex items-center gap-1.5 text-xs font-medium shadow-md shadow-indigo-600/20 transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>Launch Scrape Job</span>
      </button>
    </div>
  </div>

  <!-- Jobs Table -->
  <div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead class="bg-slate-950/60 border-b border-slate-800 text-[11px] uppercase tracking-wider text-slate-400 font-semibold">
          <tr>
            <th class="px-5 py-3.5">Query & Location</th>
            <th class="px-4 py-3.5">Status</th>
            <th class="px-4 py-3.5">Scraped</th>
            <th class="px-4 py-3.5">Enriched</th>
            <th class="px-4 py-3.5">Started At</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60 text-slate-300">
          {#if isLoading && jobs.length === 0}
            <tr>
              <td colspan="5" class="px-5 py-12 text-center text-slate-400">Loading jobs...</td>
            </tr>
          {:else if jobs.length === 0}
            <tr>
              <td colspan="5" class="px-5 py-12 text-center text-slate-400">No jobs created yet.</td>
            </tr>
          {:else}
            {#each jobs as job}
              <tr class="hover:bg-slate-800/40 transition">
                <td class="px-5 py-3.5">
                  <div class="font-semibold text-white flex items-center gap-1.5">
                    <Search class="w-3.5 h-3.5 text-indigo-400" />
                    <span>{job.query}</span>
                  </div>
                  <div class="text-[11px] text-slate-400 flex items-center gap-1 mt-0.5">
                    <MapPin class="w-3 h-3 text-slate-400" />
                    <span>{job.location}</span>
                    {#if job.category}
                      <span>•</span>
                      <span class="text-slate-400">{job.category}</span>
                    {/if}
                  </div>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <Badge variant={job.status === 'COMPLETED' ? 'success' : job.status === 'PROCESSING' ? 'warning' : job.status === 'FAILED' ? 'error' : 'default'}>
                    {#if job.status === 'COMPLETED'}
                      <CheckCircle2 class="w-3 h-3 text-emerald-400" />
                    {:else if job.status === 'PROCESSING'}
                      <RefreshCw class="w-3 h-3 animate-spin text-amber-400" />
                    {/if}
                    <span>{job.status}</span>
                  </Badge>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <span class="font-mono font-bold text-white">{job.total_scraped}</span>
                  <span class="text-[10px] text-slate-400">/ {job.max_results}</span>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  {#if job.enrich_contacts}
                    <span class="font-mono text-emerald-400 font-bold">{job.total_enriched}</span>
                    <span class="text-[10px] text-slate-400">emails</span>
                  {:else}
                    <span class="text-slate-400 text-[11px]">Disabled</span>
                  {/if}
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap text-slate-400 font-mono text-[11px]">
                  {job.created_at ? new Date(job.created_at).toLocaleString() : 'Pending'}
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <Pagination
      page={page}
      totalPages={totalPages}
      onPageChange={(newPage) => {
        page = newPage;
        fetchJobs();
      }}
    />
  </div>
</div>

<NewJobModal
  isOpen={showNewJobModal}
  onClose={() => (showNewJobModal = false)}
  onJobCreated={fetchJobs}
/>