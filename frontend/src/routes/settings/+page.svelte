<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { toast } from 'svelte-sonner';
  import { Shield, Key, Cpu, ShieldCheck, CheckCircle2, Lock } from 'lucide-svelte';

  let currentPassword = $state('');
  let newPassword = $state('');
  let isUpdatingPassword = $state(false);

  let apiKeyInput = $state('');
  let isUpdatingApiKey = $state(false);
  let systemSettings = $state<{
    has_google_api_key: boolean;
    google_api_key_masked: string;
    deduplication_mode: string;
    max_concurrent_workers: number;
  }>({
    has_google_api_key: false,
    google_api_key_masked: '',
    deduplication_mode: 'SMART_MERGE_UPSERT',
    max_concurrent_workers: 5
  });

  async function loadSettings() {
    try {
      const res = await api.get<typeof systemSettings>('/api/v1/system/settings');
      if (res) {
        systemSettings = res;
      }
    } catch (e) {
      console.error('Failed to load system settings', e);
    }
  }

  async function handleApiKeyUpdate() {
    if (!apiKeyInput.trim()) {
      toast.error('Please enter a valid Google Places API Key');
      return;
    }
    try {
      isUpdatingApiKey = true;
      const res = await api.post<{
        has_google_api_key: boolean;
        google_api_key_masked: string;
        deduplication_mode: string;
        max_concurrent_workers: number;
      }>('/api/v1/system/api-key', {
        api_key: apiKeyInput.trim()
      });
      if (res) {
        systemSettings = res;
      }
      toast.success('Google Places API Key updated and verified!');
      apiKeyInput = '';
      loadSettings();
    } catch (err: any) {
      toast.error(err.message || 'Failed to update API key');
    } finally {
      isUpdatingApiKey = false;
    }
  }

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

  onMount(() => {
    loadSettings();
  });
</script>

<div class="max-w-4xl mx-auto space-y-6 font-mono">
  <div class="pb-2 border-b border-zinc-900">
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Settings & Engine Controls</h1>
    <p class="text-xs text-zinc-500 mt-0.5">Configure live Google extraction keys, deduplication rules, and security credentials</p>
  </div>

  <!-- Extraction Engine & API Keys -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center justify-between pb-3 border-b border-zinc-800/80">
      <div class="flex items-center gap-2.5">
        <Cpu class="w-4 h-4 text-white" />
        <h2 class="text-base font-display font-light text-white">Google Places API (New) Engine</h2>
      </div>
      <div class="flex items-center gap-2">
        {#if systemSettings.has_google_api_key}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] bg-emerald-950/60 border border-emerald-800/80 text-emerald-400">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
            Key Active ({systemSettings.google_api_key_masked})
          </span>
        {:else}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] bg-amber-950/60 border border-amber-800/80 text-amber-400">
            <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
            No Key Set (Using Scraper)
          </span>
        {/if}
      </div>
    </div>

    <form onsubmit={(e) => { e.preventDefault(); handleApiKeyUpdate(); }} class="space-y-4 text-xs max-w-lg">
      <div>
        <label for="settings-api-key" class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">
          Update Google Places API Key
        </label>
        <div class="relative">
          <Key class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
          <input
            id="settings-api-key"
            type="password"
            bind:value={apiKeyInput}
            placeholder="AIzaSy..."
            class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
          />
        </div>
        <p class="text-[10px] text-zinc-500 mt-1">Requires Google Cloud "Places API (New)" enabled with Text Search permissions.</p>
      </div>

      <button
        type="submit"
        disabled={isUpdatingApiKey}
        class="px-5 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs disabled:opacity-50 transition cursor-pointer"
      >
        {isUpdatingApiKey ? 'Verifying...' : 'Save & Apply API Key'}
      </button>
    </form>
  </div>

  <!-- Database Deduplication Assurance Card -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-6 shadow-sm space-y-3">
    <div class="flex items-center gap-2.5 pb-2 border-b border-zinc-800/80">
      <ShieldCheck class="w-4 h-4 text-emerald-400" />
      <h2 class="text-base font-display font-light text-white">Database Deduplication & Normalization</h2>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-1 text-xs">
      <div class="p-3.5 rounded-xl bg-black border border-zinc-800/90 space-y-1">
        <div class="text-[10px] text-zinc-500 uppercase tracking-wider">Matching Criteria</div>
        <div class="text-zinc-200 font-medium">Place ID • Domain URL • Primary Phone • Name & City</div>
      </div>

      <div class="p-3.5 rounded-xl bg-black border border-zinc-800/90 space-y-1">
        <div class="text-[10px] text-zinc-500 uppercase tracking-wider">Conflict Action</div>
        <div class="text-emerald-400 font-medium">Merge Emails & Socials, Update Ratings/Reviews</div>
      </div>
    </div>
    <p class="text-[11px] text-zinc-500 pt-1">
      Repeated scrapes across queries and locations will never produce duplicate records in your database.
    </p>
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
        <div class="relative">
          <Lock class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
          <input
            id="settings-current-password"
            type="password"
            bind:value={currentPassword}
            required
            class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white focus:outline-hidden focus:border-zinc-400 text-xs"
          />
        </div>
      </div>

      <div>
        <label for="settings-new-password" class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">New Password</label>
        <div class="relative">
          <Key class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
          <input
            id="settings-new-password"
            type="password"
            bind:value={newPassword}
            required
            placeholder="Min 6 characters"
            class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white focus:outline-hidden focus:border-zinc-400 text-xs"
          />
        </div>
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