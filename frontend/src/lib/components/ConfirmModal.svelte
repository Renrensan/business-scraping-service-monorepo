<script lang="ts">
  import { X, AlertTriangle, Trash2 } from 'lucide-svelte';

  let {
    isOpen = false,
    title,
    message,
    confirmLabel = 'Confirm',
    variant = 'default',
    onConfirm,
    onCancel
  }: {
    isOpen: boolean;
    title: string;
    message: string;
    confirmLabel?: string;
    variant?: 'default' | 'danger';
    onConfirm: () => void;
    onCancel: () => void;
  } = $props();
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-sm shadow-2xl overflow-hidden">
      <!-- Icon + header -->
      <div class="px-6 pt-6 pb-4 flex items-start gap-4">
        <div class="shrink-0 p-2 rounded-xl {variant === 'danger' ? 'bg-rose-950 border border-rose-800' : 'bg-zinc-900 border border-zinc-700'}">
          {#if variant === 'danger'}
            <Trash2 class="w-4 h-4 text-rose-400" />
          {:else}
            <AlertTriangle class="w-4 h-4 text-amber-400" />
          {/if}
        </div>
        <div class="min-w-0">
          <h3 class="text-sm font-semibold text-white">{title}</h3>
          <p class="text-xs text-zinc-400 mt-1 leading-relaxed">{message}</p>
        </div>
      </div>

      <!-- Actions -->
      <div class="px-6 pb-5 flex items-center justify-end gap-2">
        <button
          onclick={onCancel}
          class="px-4 py-2 rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-300 hover:text-white text-xs font-medium transition cursor-pointer"
        >
          Cancel
        </button>
        <button
          onclick={onConfirm}
          class="px-4 py-2 rounded-xl text-xs font-semibold transition cursor-pointer
            {variant === 'danger'
              ? 'bg-rose-500 hover:bg-rose-400 text-white'
              : 'bg-amber-400 hover:bg-amber-300 text-black'}"
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
