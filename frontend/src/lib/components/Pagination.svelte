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

<div class="flex items-center justify-between px-4 py-3 border-t border-zinc-800/80 bg-zinc-950/70 sm:px-6">
  <div class="text-xs font-mono text-zinc-500">
    Page <span class="font-medium text-white">{page}</span> of <span class="font-medium text-white">{totalPages}</span>
  </div>
  <div class="flex items-center gap-2">
    <button
      disabled={page <= 1}
      onclick={() => onPageChange(page - 1)}
      class="px-3 py-1.5 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition cursor-pointer"
    >
      Previous
    </button>
    <button
      disabled={page >= totalPages}
      onclick={() => onPageChange(page + 1)}
      class="px-3 py-1.5 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition cursor-pointer"
    >
      Next
    </button>
  </div>
</div>

