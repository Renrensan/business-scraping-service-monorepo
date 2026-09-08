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
    { label: 'Settings & Quota', href: '/settings', icon: Settings },
  ];
</script>

<!-- Mobile Overlay -->
{#if sidebar.isMobileOpen}
  <div 
    class="fixed inset-0 bg-slate-950/80 backdrop-blur-xs z-40 md:hidden"
    onclick={() => sidebar.closeMobile()}
  ></div>
{/if}

<aside
  class="fixed md:sticky top-0 left-0 z-50 h-screen w-64 bg-slate-900 border-r border-slate-800 flex flex-col justify-between transition-transform duration-200 ease-in-out {sidebar.isMobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}"
>
  <div>
    <!-- Logo -->
    <div class="h-16 flex items-center justify-between px-6 border-b border-slate-800">
      <a href="/" class="flex items-center gap-2.5 font-bold text-base text-white tracking-tight">
        <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white shadow-md shadow-indigo-500/20">
          <Layers class="w-4 h-4" />
        </div>
        <span>BizScrape <span class="text-indigo-400 font-semibold text-xs px-1.5 py-0.5 rounded bg-indigo-500/10 border border-indigo-500/20 ml-1">PRO</span></span>
      </a>
      <button
        onclick={() => sidebar.closeMobile()}
        class="md:hidden text-slate-400 hover:text-white"
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Navigation -->
    <nav class="p-4 space-y-1.5">
      {#each navItems as item}
        {@const isActive = page.url.pathname === item.href || (item.href !== '/' && page.url.pathname.startsWith(item.href))}
        <a
          href={item.href}
          onclick={() => sidebar.closeMobile()}
          class="flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-xs font-medium transition {isActive ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'}"
        >
          <item.icon class="w-4 h-4" />
          <span>{item.label}</span>
        </a>
      {/each}
    </nav>
  </div>

  <!-- Bottom Version & Storage Info -->
  <div class="p-4 border-t border-slate-800/80 m-2 rounded-xl bg-slate-950/40 text-[11px] text-slate-400 space-y-1.5">
    <div class="flex items-center justify-between">
      <span class="text-slate-400">PostgreSQL</span>
      <span class="text-emerald-400 font-mono text-[10px]">Connected</span>
    </div>
    <div class="flex items-center justify-between">
      <span class="text-slate-400">Redis Cache</span>
      <span class="text-emerald-400 font-mono text-[10px]">Port 6380</span>
    </div>
    <div class="pt-1 text-[10px] text-slate-400 text-center">
      v1.0.0 • Go + Svelte 5
    </div>
  </div>
</aside>
