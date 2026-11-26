<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import type { ScoringProfile } from '$lib/types';
  import ScoringProfileEditor, { type ScoringOptions } from '$lib/components/ScoringProfileEditor.svelte';
  import { Trophy } from 'lucide-svelte';

  // Opens the first active profile; with none yet, offers to create one.
  let options = $state.raw<ScoringOptions | null>(null);
  let empty = $state(false);

  onMount(async () => {
    try {
      const profiles = (await api.get<ScoringProfile[]>('/api/v1/scoring/profiles')) || [];
      const pick = profiles.find((p) => p.is_active) ?? profiles[0];
      if (pick) {
        goto(`/scoring/${pick.id}`, { replaceState: true });
        return;
      }
      options = await api.get<ScoringOptions>('/api/v1/scoring/options');
      empty = true;
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to load scoring profiles'));
    }
  });
</script>

{#if empty && options}
  <div class="max-w-4xl mx-auto space-y-5">
    <div class="pb-2 border-b border-zinc-900">
      <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em] flex items-center gap-2"><Trophy class="w-5 h-5 text-zinc-400" /> Scoring</h1>
      <p class="text-xs text-zinc-500 mt-0.5">No scoring profile yet. Create one to rank leads for a product.</p>
    </div>
    <ScoringProfileEditor profile={null} {options} onSaved={(p) => goto(`/scoring/${p.id}`)} />
  </div>
{/if}
