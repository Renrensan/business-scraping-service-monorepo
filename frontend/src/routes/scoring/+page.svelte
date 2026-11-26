<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import { scoringProfiles } from '$lib/stores/scoringProfiles.svelte';

  // Opens the first active profile; with none yet, the new-profile form.
  onMount(async () => {
    try {
      await scoringProfiles.load();
      const p = scoringProfiles.first;
      goto(p ? `/scoring/${p.id}` : '/scoring/new', { replaceState: true });
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to load scoring profiles'));
    }
  });
</script>
