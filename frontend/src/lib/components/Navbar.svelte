<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import { Menu, LogOut, Key, ShieldCheck } from 'lucide-svelte';

  let { onOpenPasswordModal }: { onOpenPasswordModal: () => void } = $props();
  let showUserMenu = $state(false);
</script>

<header class="h-16 border-b border-slate-800 bg-slate-900/80 backdrop-blur-md sticky top-0 z-30 flex items-center justify-between px-4 sm:px-6">
  <div class="flex items-center gap-3">
    <button
      onclick={() => sidebar.toggleMobile()}
      class="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 md:hidden transition"
      aria-label="Toggle Navigation"
    >
      <Menu class="w-5 h-5" />
    </button>
    <div class="hidden sm:flex items-center gap-2 text-xs text-slate-400">
      <span class="inline-block w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
      <span>Scraper Service Online</span>
    </div>
  </div>

  <div class="flex items-center gap-3">
    <!-- User Profile Dropdown -->
    <div class="relative">
      <button
        onclick={() => (showUserMenu = !showUserMenu)}
        class="flex items-center gap-2.5 px-3 py-1.5 rounded-lg border border-slate-700 bg-slate-800/80 hover:bg-slate-700/80 transition text-xs font-medium text-slate-200"
      >
        <div class="w-6 h-6 rounded-full bg-indigo-500/20 text-indigo-400 flex items-center justify-center font-bold">
          {auth.user?.username?.charAt(0).toUpperCase() || 'A'}
        </div>
        <span class="hidden sm:inline">{auth.user?.username || 'Admin'}</span>
        <ShieldCheck class="w-3.5 h-3.5 text-emerald-400" />
      </button>

      {#if showUserMenu}
        <div class="absolute right-0 mt-2 w-52 bg-slate-900 border border-slate-800 rounded-xl shadow-xl py-1.5 z-50 text-xs">
          <div class="px-3 py-2 border-b border-slate-800 text-slate-400">
            <div class="font-medium text-slate-200">{auth.user?.username}</div>
            <div class="truncate text-[11px] text-slate-400">{auth.user?.email}</div>
          </div>
          <button
            onclick={() => {
              showUserMenu = false;
              onOpenPasswordModal();
            }}
            class="w-full text-left px-3 py-2 text-slate-300 hover:bg-slate-800 flex items-center gap-2 transition"
          >
            <Key class="w-3.5 h-3.5 text-slate-400" />
            Change Password
          </button>
          <button
            onclick={() => auth.logout()}
            class="w-full text-left px-3 py-2 text-rose-400 hover:bg-rose-500/10 flex items-center gap-2 transition"
          >
            <LogOut class="w-3.5 h-3.5" />
            Sign Out
          </button>
        </div>
      {/if}
    </div>
  </div>
</header>
