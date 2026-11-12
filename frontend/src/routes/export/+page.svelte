<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import type { LeadStatus } from '$lib/types';
  import KeywordMultiSelect from '$lib/components/KeywordMultiSelect.svelte';
  import LeadStatusFilter from '$lib/components/LeadStatusFilter.svelte';
  import { Download, FileSpreadsheet, Check, MessageSquare } from 'lucide-svelte';

  let search = $state('');
  let category = $state('');
  let city = $state('');
  let availableKeywords = $state<string[]>([]);
  let selectedKeywords = $state<string[]>([]);
  let selectedStatuses = $state<LeadStatus[]>([]);
  let mobileOnly = $state(true);
  let isExporting = $state(false);

  function filterParams() {
    return {
      search: search.trim() || undefined,
      category: category.trim() || undefined,
      city: city.trim() || undefined,
      keywords: selectedKeywords,
      status: selectedStatuses.length ? selectedStatuses.join(',') : undefined
    };
  }

  async function handleExport() {
    try {
      isExporting = true;
      await api.downloadCSV('/api/v1/export/csv', 'momos_leads_export.csv', filterParams());
    } finally {
      isExporting = false;
    }
  }

  async function handleWAExport() {
    try {
      isExporting = true;
      const headers = await api.downloadCSV(
        '/api/v1/export/wa-contacts',
        'momos_wa_contacts.csv',
        { ...filterParams(), mobile_only: mobileOnly },
        null
      );
      if (!headers) return;
      const exported = Number(headers.get('X-Export-Count') ?? 0);
      const skipped = Number(headers.get('X-Export-Skipped') ?? 0);
      if (exported === 0) toast.error('No leads with a usable WhatsApp number match these filters');
      else toast.success(`Exported ${exported.toLocaleString()} contacts` + (skipped ? ` · ${skipped.toLocaleString()} skipped` : ''));
    } finally {
      isExporting = false;
    }
  }

  onMount(async () => {
    try {
      const res = await api.get<string[]>('/api/v1/businesses/keywords');
      if (Array.isArray(res)) availableKeywords = res.filter(Boolean);
    } catch {
      // silently ignore
    }
  });
</script>

<div class="max-w-4xl mx-auto space-y-6">
  <div class="pb-2 border-b border-zinc-900">
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Export Leads</h1>
    <p class="text-xs text-zinc-500 mt-0.5">Download your scraped business listings and contact records as a CSV spreadsheet</p>
  </div>

  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-6 shadow-sm space-y-6">
    <div class="flex items-center gap-3 pb-5 border-b border-zinc-800/80">
      <div class="w-10 h-10 rounded-xl bg-zinc-900 text-white flex items-center justify-center border border-zinc-800">
        <FileSpreadsheet class="w-5 h-5" />
      </div>
      <div>
        <h2 class="text-base font-semibold text-white">Full Database CSV Export</h2>
        <p class="text-xs text-zinc-500">Download formatted spreadsheet compatible with Excel, Google Sheets, and CRMs</p>
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
      <div>
        <label for="export-search" class="block text-zinc-400 mb-1.5 uppercase text-[11px] font-semibold tracking-wider">Search Query Filter</label>
        <input
          id="export-search"
          type="text"
          bind:value={search}
          placeholder="Filter by keyword..."
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <div>
        <label for="export-category" class="block text-zinc-400 mb-1.5 uppercase text-[11px] font-semibold tracking-wider">Category Filter</label>
        <input
          id="export-category"
          type="text"
          bind:value={category}
          placeholder="e.g. Villas & Resorts, Cafes"
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <div>
        <label for="export-city" class="block text-zinc-400 mb-1.5 uppercase text-[11px] font-semibold tracking-wider">City / Region Filter</label>
        <input
          id="export-city"
          type="text"
          bind:value={city}
          placeholder="e.g. Canggu, Seminyak, Ubud"
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs items-start">
      <div>
        <span class="block text-zinc-400 mb-1.5 uppercase text-[11px] font-semibold tracking-wider">Scrape Keywords</span>
        <KeywordMultiSelect options={availableKeywords} selected={selectedKeywords} onChange={(k) => (selectedKeywords = k)} />
      </div>
      <div class="sm:col-span-2 sm:pt-6">
        <LeadStatusFilter selected={selectedStatuses} onChange={(s) => (selectedStatuses = s)} />
      </div>
    </div>

    <div class="p-4 rounded-xl bg-black border border-zinc-800 space-y-2.5">
      <div class="text-[11px] uppercase text-zinc-500 font-semibold tracking-wider">Included Columns in Export</div>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs text-zinc-300">
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Business Name</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Category</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Primary Phone</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Website URL</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Verified Emails</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Full Address</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Star Rating</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Review Count</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> LinkedIn Link</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Instagram Link</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Facebook Link</div>
        <div class="flex items-center gap-1.5"><Check class="w-3.5 h-3.5 text-emerald-400" /> Google Maps URL</div>
      </div>
    </div>

    <div class="pt-2 flex flex-col sm:flex-row sm:items-center justify-end gap-3">
      <label class="inline-flex items-center gap-2 text-xs text-zinc-400 cursor-pointer select-none sm:mr-auto">
        <input type="checkbox" bind:checked={mobileOnly} class="w-3.5 h-3.5 accent-emerald-500 cursor-pointer" />
        WhatsApp export: mobile numbers only
      </label>
      <button
        onclick={handleWAExport}
        disabled={isExporting}
        class="px-5 py-2.5 rounded-xl border border-emerald-800 bg-emerald-950/60 hover:bg-emerald-900/70 text-emerald-300 font-semibold text-xs flex items-center justify-center gap-2 disabled:opacity-50 transition cursor-pointer"
        title="Google Contacts CSV — import at contacts.google.com and the numbers appear in WhatsApp"
      >
        <MessageSquare class="w-4 h-4" />
        <span>WhatsApp Contacts</span>
      </button>
      <button
        onclick={handleExport}
        disabled={isExporting}
        class="px-6 py-2.5 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs flex items-center justify-center gap-2 shadow-sm disabled:opacity-50 transition cursor-pointer"
      >
        <Download class="w-4 h-4" />
        <span>{isExporting ? 'Downloading...' : 'Download CSV'}</span>
      </button>
    </div>
  </div>
</div>