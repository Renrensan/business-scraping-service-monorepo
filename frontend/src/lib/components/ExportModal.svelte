<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { X, Download, FileSpreadsheet, Check, MessageSquare } from 'lucide-svelte';

  let {
    isOpen = false,
    onClose,
    filterParams = {}
  }: {
    isOpen: boolean;
    onClose: () => void;
    filterParams?: Record<string, any>;
  } = $props();

  type ExportFormat = 'full' | 'wa';

  let format = $state<ExportFormat>('full');
  let mobileOnly = $state(true);
  let statusPrefix = $state(false);
  let isExporting = $state(false);

  async function handleDownload() {
    try {
      isExporting = true;
      if (format === 'full') {
        await api.downloadCSV('/api/v1/export/csv', 'momos_leads_export.csv', filterParams);
      } else {
        const headers = await api.downloadCSV(
          '/api/v1/export/wa-contacts',
          'momos_wa_contacts.csv',
          { ...filterParams, mobile_only: mobileOnly, status_prefix: statusPrefix || undefined },
          null
        );
        if (!headers) return;
        const exported = Number(headers.get('X-Export-Count') ?? 0);
        const skipped = Number(headers.get('X-Export-Skipped') ?? 0);
        if (exported === 0) {
          toast.error('No leads with a usable WhatsApp number match these filters');
        } else {
          toast.success(
            `Exported ${exported.toLocaleString()} contacts` +
              (skipped ? ` · ${skipped.toLocaleString()} skipped (no ${mobileOnly ? 'mobile' : 'valid'} number or duplicate)` : '')
          );
        }
      }
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
        <!-- Format picker -->
        <div class="grid grid-cols-2 gap-2" role="radiogroup" aria-label="Export format">
          <button
            type="button"
            role="radio"
            aria-checked={format === 'full'}
            onclick={() => (format = 'full')}
            class="p-3 rounded-xl border text-left transition cursor-pointer {format === 'full'
              ? 'bg-white/10 border-white/30 text-white'
              : 'bg-black border-zinc-800 text-zinc-400 hover:border-zinc-700'}"
          >
            <FileSpreadsheet class="w-4 h-4 mb-1.5" />
            <div class="font-semibold">Full data CSV</div>
            <div class="text-[10px] text-zinc-500 mt-0.5">Every column, for Excel / Sheets / CRMs</div>
          </button>
          <button
            type="button"
            role="radio"
            aria-checked={format === 'wa'}
            onclick={() => (format = 'wa')}
            class="p-3 rounded-xl border text-left transition cursor-pointer {format === 'wa'
              ? 'bg-emerald-950/60 border-emerald-700 text-white'
              : 'bg-black border-zinc-800 text-zinc-400 hover:border-zinc-700'}"
          >
            <MessageSquare class="w-4 h-4 mb-1.5 text-emerald-400" />
            <div class="font-semibold">WhatsApp contacts</div>
            <div class="text-[10px] text-zinc-500 mt-0.5">Google Contacts CSV, numbers as +62…</div>
          </button>
        </div>

        {#if format === 'full'}
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
              <div class="flex items-center gap-1.5"><Check class="w-3 h-3 text-emerald-400" /> Lead Status</div>
            </div>
          </div>
        {:else}
          <div class="space-y-2">
            <label class="flex items-start gap-2.5 p-2.5 rounded-xl bg-black border border-zinc-800 cursor-pointer select-none">
              <input type="checkbox" bind:checked={mobileOnly} class="mt-0.5 w-3.5 h-3.5 accent-emerald-500 cursor-pointer" />
              <span>
                <span class="text-zinc-200 font-medium">Mobile numbers only</span>
                <span class="block text-[10px] text-zinc-500">Skip landlines (e.g. 0361…) — they can't have WhatsApp. Uses a mobile found on the website if the main number is a landline.</span>
              </span>
            </label>
            <label class="flex items-start gap-2.5 p-2.5 rounded-xl bg-black border border-zinc-800 cursor-pointer select-none">
              <input type="checkbox" bind:checked={statusPrefix} class="mt-0.5 w-3.5 h-3.5 accent-emerald-500 cursor-pointer" />
              <span>
                <span class="text-zinc-200 font-medium">Add status to names</span>
                <span class="block text-[10px] text-zinc-500">e.g. "[HOT] Kopi Canggu", so the status shows in your WhatsApp chat list</span>
              </span>
            </label>
          </div>

          <div class="p-3 rounded-xl bg-emerald-950/20 border border-emerald-900/60 text-[11px] text-zinc-300 space-y-1">
            <div class="font-semibold text-emerald-400">How to import</div>
            <ol class="list-decimal list-inside space-y-0.5 text-zinc-400">
              <li>Open <a href="https://contacts.google.com" target="_blank" rel="noreferrer" class="text-white underline">contacts.google.com</a> with the account on your phone</li>
              <li>Click <span class="text-zinc-200">Import</span> and choose the downloaded CSV</li>
              <li>Contacts sync to the phone and appear in WhatsApp, grouped under the "Momos Leads" label</li>
            </ol>
          </div>
        {/if}

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
            <span>{isExporting ? 'Streaming...' : format === 'wa' ? 'Download WA Contacts' : 'Download CSV'}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
