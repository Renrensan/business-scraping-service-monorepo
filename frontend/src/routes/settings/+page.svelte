<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { toast } from 'svelte-sonner';
  import { Shield, Key, CheckCircle2 } from 'lucide-svelte';

  let currentPassword = $state('');
  let newPassword = $state('');
  let isUpdatingPassword = $state(false);

  async function handlePasswordChange() {
    if (!newPassword || newPassword.length < 6) {
      toast.error('New password must be at least 6 characters');
      return;
    }
    try {
      isUpdatingPassword = true;
      await api.post('/api/v1/auth/change-password', {
        current_password: currentPassword,
        new_password: newPassword
      });
      toast.success('Password updated successfully');
      currentPassword = '';
      newPassword = '';
    } catch (err: any) {
      toast.error(err.message || 'Failed to update password');
    } finally {
      isUpdatingPassword = false;
    }
  }
</script>

<div class="max-w-4xl mx-auto space-y-6 font-mono">
  <div class="pb-2 border-b border-zinc-900">
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Settings</h1>
    <p class="text-xs text-zinc-500 mt-0.5">Manage administrator credentials and account preferences</p>
  </div>

  <!-- Admin Account Security -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center gap-2.5 pb-3 border-b border-zinc-800/80">
      <Shield class="w-4 h-4 text-white" />
      <h2 class="text-base font-display font-light text-white">Account Security & Credentials</h2>
    </div>

    <form onsubmit={(e) => { e.preventDefault(); handlePasswordChange(); }} class="space-y-4 text-xs max-w-md">
      <div>
        <label for="settings-current-password" class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">Current Password</label>
        <input
          id="settings-current-password"
          type="password"
          bind:value={currentPassword}
          required
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <div>
        <label for="settings-new-password" class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">New Password</label>
        <input
          id="settings-new-password"
          type="password"
          bind:value={newPassword}
          required
          placeholder="Min 6 characters"
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <button
        type="submit"
        disabled={isUpdatingPassword}
        class="px-5 py-2.5 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs disabled:opacity-50 transition cursor-pointer"
      >
        {isUpdatingPassword ? 'Saving...' : 'Update Password'}
      </button>
    </form>
  </div>
</div>