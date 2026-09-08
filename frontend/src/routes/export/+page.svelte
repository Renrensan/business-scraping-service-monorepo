<script lang="ts">
  import { api } from '$lib/api/client';
  import { Download, FileSpreadsheet, Check, Sparkles } from 'lucide-svelte';

  let search = $state('');
  let category = $state('');
  let city = $state('');
  let isExporting = $state(false);

  async function handleExport() {
    try {
      isExporting = true;
      await api.downloadCSV('/api/v1/export/csv', 'business_leads_export.csv', {
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
  <div>
    <h1 class="text-xl font-bold text-white tracking-tight">Bulk Export Center</h1>
    <p class="text-xs text-slate-400 mt-1">Export raw business leads and verified email contact lists to CSV spreadsheets</p>
  </div>

  <div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-sm space-y-6">
    <div class="flex items-center gap-3 pb-5 border-b border-slate-800">
      <div class="w-10 h-10 rounded-xl bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
        <FileSpreadsheet class="w-5 h-5" />
      </div>
      <div>
        <h2 class="text-sm font-bold text-white">Full Database CSV Export</h2>
        <p class="text-xs text-slate-400">Download formatted CSV compatible with Excel, Google Sheets, and CRM tools</p>
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
      <div>
        <label class="block font-medium text-slate-300 mb-1.5">Optional Search Query</label>
        <input
          type="text"
          bind:value={search}
          placeholder="Filter by name..."
          class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
        />
      </div>

      <div>
        <label class="block font-medium text-slate-300 mb-1.5">Optional Category</label>
        <input
          type="text"
          bind:value={category}
          placeholder="e.g. Coffee Shop"
          class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
        />
      </div>

      <div>
        <label class="block font-medium text-slate-300 mb-1.5">Optional City</label>
        <input
          type="text"
          bind:value={city}
          placeholder="e.g. Seattle"
          class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
        />
      </div>
    </div>

    <div class="p-4 rounded-xl bg-slate-950/60 border border-slate-800 space-y-2.5">
      <div class="text-[11px] font-semibold uppercase text-slate-400">Included Columns in CSV</div>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-[11px] text-slate-300">
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
        class="px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs flex items-center gap-2 shadow-lg shadow-emerald-600/20 disabled:opacity-50 transition cursor-pointer"
      >
        <Download class="w-4 h-4" />
        <span>{isExporting ? 'Generating Spreadsheet...' : 'Download CSV Export'}</span>
      </button>
    </div>
  </div>
</div>