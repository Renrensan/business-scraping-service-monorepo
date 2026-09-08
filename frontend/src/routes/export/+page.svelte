<script lang="ts">
  import { api } from '$lib/api/client';
  import { Download, FileSpreadsheet, Check } from 'lucide-svelte';

  let search = $state('');
  let category = $state('');
  let city = $state('');
  let isExporting = $state(false);

  async function handleExport() {
    try {
      isExporting = true;
      await api.downloadCSV('/api/v1/export/csv', 'momos_leads_export.csv', {
        search: search.trim() || undefined,
        category: category.trim() || undefined,
        city: city.trim() || undefined
      });
    } finally {
      isExporting = false;
    }
  }
</script>

<div class="max-w-4xl mx-auto space-y-6">
  <div class="pb-2 border-b border-zinc-900">
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Export Center</h1>
    <p class="text-xs font-mono text-zinc-500 mt-0.5">Stream consolidated business leads, verified contacts, and social handles directly to CSV</p>
  </div>

  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-6 shadow-sm space-y-6">
    <div class="flex items-center gap-3 pb-5 border-b border-zinc-800/80">
      <div class="w-10 h-10 rounded-xl bg-zinc-900 text-white flex items-center justify-center border border-zinc-800">
        <FileSpreadsheet class="w-5 h-5" />
      </div>
      <div>
        <h2 class="text-base font-display font-light text-white">Full Database CSV Export</h2>
        <p class="text-xs font-mono text-zinc-500">Fast streaming format compatible with Excel, Google Sheets, Apollo, and CRMs</p>
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs font-mono">
      <div>
        <label for="export-search" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">Search Query Filter</label>
        <input
          id="export-search"
          type="text"
          bind:value={search}
          placeholder="Filter by keyword..."
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <div>
        <label for="export-category" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">Category Filter</label>
        <input
          id="export-category"
          type="text"
          bind:value={category}
          placeholder="e.g. Coffee Shop"
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <div>
        <label for="export-city" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">City / Region Filter</label>
        <input
          id="export-city"
          type="text"
          bind:value={city}
          placeholder="e.g. Seattle"
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>
    </div>

    <div class="p-4 rounded-xl bg-black border border-zinc-800 space-y-2.5 font-mono">
      <div class="text-[10px] uppercase text-zinc-500 tracking-wider">Included Columns in Export</div>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-[11px] text-zinc-300">
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Business Name</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Category</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Primary Phone</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Website URL</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Verified Emails</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Full Address</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Star Rating</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Review Count</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> LinkedIn Link</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Instagram Link</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Facebook Link</div>
        <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Google Maps URL</div>
      </div>
    </div>

    <div class="pt-2 flex justify-end">
      <button
        onclick={handleExport}
        disabled={isExporting}
        class="px-6 py-2.5 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs font-mono flex items-center gap-2 shadow-sm disabled:opacity-50 transition cursor-pointer"
      >
        <Download class="w-4 h-4" />
        <span>{isExporting ? 'Generating Stream...' : 'Download CSV Export'}</span>
      </button>
    </div>
  </div>
</div>