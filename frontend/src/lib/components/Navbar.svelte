<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import { Menu, LogOut, Key, ShieldCheck } from 'lucide-svelte';

  let { onOpenPasswordModal }: { onOpenPasswordModal: () => void } = $props();
  let showUserMenu = $state(false);
</script>

<header class="h-16 border-b border-zinc-800/80 bg-black/80 backdrop-blur-md sticky top-0 z-30 flex items-center justify-between px-4 sm:px-6">
  <div class="flex items-center gap-3">
    <button
      onclick={() => sidebar.toggleMobile()}
      class="p-2 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-900 md:hidden transition cursor-pointer"
      aria-label="Toggle Navigation"
    >
      <Menu class="w-5 h-5" />
    </button>
    <div class="hidden sm:flex items-center gap-2 text-xs font-mono text-zinc-400">
      <span class="inline-block w-2 h-2 rounded-full bg-emerald-400"></span>
      <span class="text-zinc-300">SYSTEM ONLINE</span>
      <span class="text-zinc-600">•</span>
      <span class="text-zinc-500">INGESTION ENGINE ACTIVE</span>
    </div>
  </div>

  <div class="flex items-center gap-3">
    <!-- User Profile Dropdown -->
    <div class="relative">
      <button
        onclick={() => (showUserMenu = !showUserMenu)}
        class="flex items-center gap-2.5 px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/80 hover:bg-zinc-800 transition text-xs font-mono text-zinc-200 cursor-pointer"
      >
        <div class="w-5 h-5 rounded bg-zinc-800 text-white flex items-center justify-center font-bold text-[10px] border border-zinc-700">
          {auth.user?.username?.charAt(0).toUpperCase() || 'A'}
        </div>
        <span class="hidden sm:inline">{auth.user?.username || 'admin'}</span>
        <ShieldCheck class="w-3.5 h-3.5 text-emerald-400" />
      </button>

      {#if showUserMenu}
        <div class="absolute right-0 mt-2 w-52 bg-zinc-950 border border-zinc-800 rounded-xl shadow-2xl py-1.5 z-50 text-xs font-mono">
          <div class="px-3.5 py-2.5 border-b border-zinc-800/80 text-zinc-400">
            <div class="font-medium text-white">{auth.user?.username}</div>
            <div class="truncate text-[10px] text-zinc-500 mt-0.5">{auth.user?.email}</div>
          </div>
          <button
            onclick={() => {
              showUserMenu = false;
              onOpenPasswordModal();
            }}
            class="w-full text-left px-3.5 py-2 text-zinc-300 hover:bg-zinc-900 hover:text-white flex items-center gap-2 transition cursor-pointer"
          >
            <Key class="w-3.5 h-3.5 text-zinc-400" />
            Change Password
          </button>
          <button
            onclick={() => auth.logout()}
            class="w-full text-left px-3.5 py-2 text-rose-400 hover:bg-rose-950/30 flex items-center gap-2 transition cursor-pointer"
          >
            <LogOut class="w-3.5 h-3.5" />
            Sign Out
          </button>
        </div>
      {/if}
    </div>
  </div>
</header>

