<script lang="ts">
  let {
    page = 1,
    totalPages = 1,
    onPageChange
  }: {
    page: number;
    totalPages: number;
    onPageChange: (newPage: number) => void;
  } = $props();
</script>

<div class="flex items-center justify-between px-4 py-3 border-t border-slate-800 bg-slate-900/50 sm:px-6">
  <div class="text-xs text-slate-400">
    Page <span class="font-medium text-slate-200">{page}</span> of <span class="font-medium text-slate-200">{totalPages}</span>
  </div>
  <div class="flex items-center gap-2">
    <button
      disabled={page <= 1}
      onclick={() => onPageChange(page - 1)}
      class="px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-700 bg-slate-800 text-slate-300 hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed transition"
    >
      Previous
    </button>
    <button
      disabled={page >= totalPages}
      onclick={() => onPageChange(page + 1)}
      class="px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-700 bg-slate-800 text-slate-300 hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed transition"
    >
      Next
    </button>
  </div>
</div>
