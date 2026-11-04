<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { 
    Shield, Key, Cpu, ShieldCheck, CheckCircle2, Lock, 
    Calendar, KeyRound, ArrowRight, Save, Eye, EyeOff,
    Zap, Building2
  } from 'lucide-svelte';

  let currentPassword = $state('');
  let newPassword = $state('');
  let showCurrentPassword = $state(false);
  let showNewPassword = $state(false);
  let isUpdatingPassword = $state(false);

  let apiKeyInput = $state('');
  let isUpdatingApiKey = $state(false);
  let showApiKey = $state(false);
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
      const res = await api.post<typeof systemSettings>('/api/v1/system/api-key', {
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

<div class="max-w-4xl mx-auto space-y-6">
  <!-- Page Header -->
  <div class="pb-2 border-b border-zinc-900">
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">System Settings</h1>
    <p class="text-xs text-zinc-500 mt-0.5">Manage Google extraction API keys, scraping engine parameters, and administrator security</p>
  </div>

  <!-- Dedicated Links to Modules -->
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
    <a
      href="/scraper"
      class="p-5 rounded-2xl bg-zinc-950 border border-zinc-800 hover:border-zinc-700 transition group flex items-center justify-between"
    >
      <div class="flex items-center gap-3.5">
        <div class="w-10 h-10 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center text-zinc-300 group-hover:text-white">
          <Zap class="w-5 h-5" />
        </div>
        <div>
          <div class="text-sm font-medium text-white group-hover:text-zinc-200">Scraper & Keywords</div>
          <div class="text-xs text-zinc-500">Manage queries, auto-schedule cycles & instant scrape</div>
        </div>
      </div>
      <ArrowRight class="w-4 h-4 text-zinc-500 group-hover:text-white group-hover:translate-x-1 transition" />
    </a>

    <a
      href="/businesses"
      class="p-5 rounded-2xl bg-zinc-950 border border-zinc-800 hover:border-zinc-700 transition group flex items-center justify-between"
    >
      <div class="flex items-center gap-3.5">
        <div class="w-10 h-10 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center text-zinc-300 group-hover:text-white">
          <Building2 class="w-5 h-5" />
        </div>
        <div>
          <div class="text-sm font-medium text-white group-hover:text-zinc-200">Leads Database</div>
          <div class="text-xs text-zinc-500">View scraped business listings, emails & filters</div>
        </div>
      </div>
      <ArrowRight class="w-4 h-4 text-zinc-500 group-hover:text-white group-hover:translate-x-1 transition" />
    </a>
  </div>

  <!-- Google Places API Key Card -->
  <div class="bg-zinc-950 border border-zinc-800 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center justify-between pb-3 border-b border-zinc-800">
      <div class="flex items-center gap-2.5">
        <Key class="w-4 h-4 text-white" />
        <h2 class="text-base font-medium text-white">Google Places API Key</h2>
      </div>
      <div>
        {#if systemSettings.has_google_api_key}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-950/60 border border-emerald-800/80 text-emerald-400">
            <CheckCircle2 class="w-3.5 h-3.5" />
            Configured & Ready
          </span>
        {:else}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-950/60 border border-amber-800/80 text-amber-400">
            API Key Required
          </span>
        {/if}
      </div>
    </div>

    <p class="text-xs text-zinc-400 leading-relaxed">
      Google Maps Places TextSearch API key used for automatic background scraping and on-demand search queries.
    </p>

    {#if systemSettings.google_api_key_masked}
      <div class="p-3.5 rounded-xl bg-black border border-zinc-800 flex items-center justify-between">
        <div>
          <div class="text-[11px] text-zinc-500 uppercase tracking-wider font-semibold">Active Key (Masked)</div>
          <div class="text-xs font-mono text-zinc-300 mt-0.5">{systemSettings.google_api_key_masked}</div>
        </div>
        <span class="text-xs text-emerald-400 font-medium flex items-center gap-1">
          <ShieldCheck class="w-3.5 h-3.5" />
          Encrypted
        </span>
      </div>
    {/if}

    <form onsubmit={(e) => { e.preventDefault(); handleApiKeyUpdate(); }} class="space-y-3">
      <div>
        <label for="api-key-input" class="block text-xs font-medium text-zinc-400 mb-1.5">
          {systemSettings.has_google_api_key ? 'Update Google Places API Key' : 'Enter Google Places API Key'}
        </label>
        <div class="flex gap-2">
          <input
            id="api-key-input"
            type={showApiKey ? 'text' : 'password'}
            bind:value={apiKeyInput}
            placeholder="AIzaSy..."
            class="flex-1 bg-black border border-zinc-800 rounded-xl px-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 font-mono text-xs"
          />
          <button
            type="button"
            onclick={() => showApiKey = !showApiKey}
            class="px-3 py-2 rounded-xl border border-zinc-800 bg-black text-zinc-400 hover:text-white transition cursor-pointer"
            title={showApiKey ? 'Hide Key' : 'Show Key'}
          >
            {#if showApiKey}
              <EyeOff class="w-4 h-4" />
            {:else}
              <Eye class="w-4 h-4" />
            {/if}
          </button>
          <button
            type="submit"
            disabled={isUpdatingApiKey || !apiKeyInput.trim()}
            class="px-4 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs flex items-center gap-1.5 shadow-sm disabled:opacity-50 transition cursor-pointer"
          >
            <Save class="w-3.5 h-3.5" />
            <span>{isUpdatingApiKey ? 'Saving...' : 'Save Key'}</span>
          </button>
        </div>
      </div>
    </form>
  </div>

  <!-- Engine Performance & Concurrency Card -->
  <div class="bg-zinc-950 border border-zinc-800 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center gap-2.5 pb-3 border-b border-zinc-800">
      <Cpu class="w-4 h-4 text-white" />
      <h2 class="text-base font-medium text-white">Scraper Engine & Storage Strategy</h2>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
      <div class="p-4 rounded-xl bg-black border border-zinc-800 space-y-1">
        <div class="text-[11px] text-zinc-500 uppercase tracking-wider font-semibold">Deduplication Mode</div>
        <div class="text-sm font-medium text-white">{systemSettings.deduplication_mode}</div>
        <div class="text-zinc-400 text-xs mt-1">
          Upserts records by Google Place ID and business website domain to prevent duplicates.
        </div>
      </div>

      <div class="p-4 rounded-xl bg-black border border-zinc-800 space-y-1">
        <div class="text-[11px] text-zinc-500 uppercase tracking-wider font-semibold">Max Concurrent Workers</div>
        <div class="text-sm font-mono font-medium text-white">{systemSettings.max_concurrent_workers} parallel workers</div>
        <div class="text-zinc-400 text-xs mt-1">
          Scraping and contact enrichment tasks run in parallel goroutines.
        </div>
      </div>
    </div>
  </div>

  <!-- Account Security Card -->
  <div class="bg-zinc-950 border border-zinc-800 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center gap-2.5 pb-3 border-b border-zinc-800">
      <Shield class="w-4 h-4 text-white" />
      <h2 class="text-base font-medium text-white">Administrator Security</h2>
    </div>

    <p class="text-xs text-zinc-400">
      Update your administrator password for the MŌMOS web portal.
    </p>

    <form onsubmit={(e) => { e.preventDefault(); handlePasswordChange(); }} class="space-y-3">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label for="current-password" class="block text-xs font-medium text-zinc-400 mb-1.5">Current Password</label>
          <div class="relative">
            <input
              id="current-password"
              type={showCurrentPassword ? 'text' : 'password'}
              bind:value={currentPassword}
              placeholder="••••••••"
              class="w-full bg-black border border-zinc-800 rounded-xl pl-3.5 pr-10 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
            />
            <button
              type="button"
              onclick={() => (showCurrentPassword = !showCurrentPassword)}
              class="absolute right-3 top-2 text-zinc-500 hover:text-zinc-300 transition cursor-pointer"
              aria-label={showCurrentPassword ? 'Hide current password' : 'Show current password'}
              tabindex="-1"
            >
              {#if showCurrentPassword}
                <EyeOff class="w-4 h-4" />
              {:else}
                <Eye class="w-4 h-4" />
              {/if}
            </button>
          </div>
        </div>
        <div>
          <label for="new-password" class="block text-xs font-medium text-zinc-400 mb-1.5">New Password (Min 6 chars)</label>
          <div class="relative">
            <input
              id="new-password"
              type={showNewPassword ? 'text' : 'password'}
              bind:value={newPassword}
              placeholder="••••••••"
              class="w-full bg-black border border-zinc-800 rounded-xl pl-3.5 pr-10 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
            />
            <button
              type="button"
              onclick={() => (showNewPassword = !showNewPassword)}
              class="absolute right-3 top-2 text-zinc-500 hover:text-zinc-300 transition cursor-pointer"
              aria-label={showNewPassword ? 'Hide new password' : 'Show new password'}
              tabindex="-1"
            >
              {#if showNewPassword}
                <EyeOff class="w-4 h-4" />
              {:else}
                <Eye class="w-4 h-4" />
              {/if}
            </button>
          </div>
        </div>
      </div>

      <div class="flex justify-end pt-2">
        <button
          type="submit"
          disabled={isUpdatingPassword || !newPassword}
          class="px-5 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs flex items-center gap-1.5 shadow-sm disabled:opacity-50 transition cursor-pointer"
        >
          <Lock class="w-3.5 h-3.5" />
          <span>{isUpdatingPassword ? 'Updating...' : 'Update Password'}</span>
        </button>
      </div>
    </form>
  </div>
</div>
