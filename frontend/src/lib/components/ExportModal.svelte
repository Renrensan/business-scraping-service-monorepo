<script lang="ts">
  import { api } from '$lib/api/client';
  import { X, Download, FileSpreadsheet, Check } from 'lucide-svelte';

  let {
    isOpen = false,
    onClose,
    filterParams = {}
  }: {
    isOpen: boolean;
    onClose: () => void;
    filterParams?: Record<string, any>;
  } = $props();

  let isExporting = $state(false);

  async function handleDownload() {
    try {
      isExporting = true;
      await api.downloadCSV('/api/v1/export/csv', 'momos_leads_export.csv', filterParams);
      onClose();
    } finally {
      isExporting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl">
      <div class="flex items-center justify-between px-6 py-4 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <FileSpreadsheet class="w-4 h-4 text-emerald-400" />
          <h2 class="text-base font-semibold text-white">Export Filtered Leads</h2>
        </div>
        <button onclick={onClose} class="text-zinc-500 hover:text-white transition cursor-pointer" aria-label="Close export modal">
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-6 space-y-4 text-xs">
        <p class="text-zinc-300 leading-relaxed">
          Stream a consolidated CSV spreadsheet of all business leads and enriched contact records matching active filters.
        </p>

        <div class="p-3.5 rounded-xl bg-black border border-zinc-800 space-y-2">
          <div class="text-[10px] uppercase text-zinc-500 tracking-wider">Exported Columns</div>
          <div class="grid grid-cols-2 gap-1.5 text-[11px] text-zinc-300">
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Business Name</div>
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Primary Phone</div>
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Category</div>
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Extracted Emails</div>
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Full Address</div>
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Social Links</div>
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Ratings & Reviews</div>
            <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Coordinates (Lat/Lng)</div>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-zinc-800">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 hover:text-white transition cursor-pointer"
          >
            Cancel
          </button>
          <button
            onclick={handleDownload}
            disabled={isExporting}
            class="px-5 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold flex items-center gap-1.5 shadow-sm disabled:opacity-50 transition cursor-pointer"
          >
            <Download class="w-3.5 h-3.5" />
            <span>{isExporting ? 'Streaming...' : 'Download CSV'}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

