<script lang="ts">
  import { page } from '$app/state';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import { 
    LayoutDashboard, 
    Building2, 
    Zap, 
    Download, 
    Settings, 
    X,
    Layers
  } from 'lucide-svelte';

  const navItems = [
    { label: 'Overview', href: '/', icon: LayoutDashboard },
    { label: 'Leads Explorer', href: '/businesses', icon: Building2 },
    { label: 'Scrape Jobs', href: '/jobs', icon: Zap },
    { label: 'Bulk Export', href: '/export', icon: Download },
    { label: 'System & Security', href: '/settings', icon: Settings },
  ];
</script>

<!-- Mobile Overlay -->
{#if sidebar.isMobileOpen}
  <div 
    class="fixed inset-0 bg-black/80 backdrop-blur-xs z-40 md:hidden"
    onclick={() => sidebar.closeMobile()}
  ></div>
{/if}

<aside
  class="fixed md:sticky top-0 left-0 z-50 h-screen w-64 bg-black border-r border-zinc-800/90 flex flex-col justify-between transition-transform duration-200 ease-in-out {sidebar.isMobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}"
>
  <div>
    <!-- Brand Wordmark Header -->
    <div class="h-20 flex items-center justify-between px-6 border-b border-zinc-800/80">
      <a href="/" class="flex flex-col group">
        <div class="flex items-center gap-2">
          <span class="text-2xl font-display font-light text-white tracking-[-0.05em] group-hover:text-zinc-200 transition">
            MŌMOS
          </span>
          <span class="text-[9px] font-mono uppercase px-1.5 py-0.5 rounded bg-zinc-900 border border-zinc-700 text-zinc-300">
            ENGINE
          </span>
        </div>
        <span class="text-[10px] font-mono text-zinc-500 uppercase tracking-wider mt-0.5">
          LEAD INTELLIGENCE
        </span>
      </a>
      <button
        onclick={() => sidebar.closeMobile()}
        class="md:hidden text-zinc-400 hover:text-white"
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Navigation -->
    <nav class="p-3.5 space-y-1">
      {#each navItems as item}
        {@const isActive = page.url.pathname === item.href || (item.href !== '/' && page.url.pathname.startsWith(item.href))}
        <a
          href={item.href}
          onclick={() => sidebar.closeMobile()}
          class="flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-xs font-mono transition {isActive ? 'bg-white text-black font-medium shadow-sm' : 'text-zinc-400 hover:text-white hover:bg-zinc-900/80'}"
        >
          <item.icon class="w-4 h-4 {isActive ? 'text-black' : 'text-zinc-400'}" />
          <span>{item.label}</span>
        </a>
      {/each}
    </nav>
  </div>

  <!-- Bottom Version & Storage Info -->
  <div class="p-4 border-t border-zinc-800/80 m-3 rounded-xl bg-zinc-950/80 border text-[11px] font-mono text-zinc-500 space-y-2">
    <div class="flex items-center justify-between">
      <span>PostgreSQL</span>
      <span class="text-emerald-400 font-mono text-[10px]">Connected</span>
    </div>
    <div class="flex items-center justify-between">
      <span>Redis Cache</span>
      <span class="text-emerald-400 font-mono text-[10px]">Port 6380</span>
    </div>
    <div class="pt-1.5 border-t border-zinc-900 text-[10px] text-zinc-600 text-center uppercase tracking-wider">
      MŌMOS v1.0.0
    </div>
  </div>
</aside>

