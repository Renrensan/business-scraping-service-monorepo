<script lang="ts">
  import { page } from '$app/state';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import { 
    LayoutDashboard, 
    Database,
    UserCheck,
    Zap, 
    Download, 
    Settings, 
    X,
    PanelLeftClose,
    PanelLeftOpen
  } from 'lucide-svelte';

  const navItems = [
    { label: 'Overview', href: '/', icon: LayoutDashboard },
    { label: 'Scraper', href: '/scraper', icon: Zap },
    { label: 'Raw Data', href: '/businesses', icon: Database },
    { label: 'Leads', href: '/leads', icon: UserCheck },
    { label: 'Export', href: '/export', icon: Download },
    { label: 'Settings', href: '/settings', icon: Settings },
  ];
</script>

<!-- Mobile Overlay -->
{#if sidebar.isMobileOpen}
  <button 
    type="button"
    aria-label="Close Sidebar"
    class="fixed inset-0 bg-black/80 backdrop-blur-xs z-40 md:hidden w-full h-full cursor-default"
    onclick={() => sidebar.closeMobile()}
  ></button>
{/if}

<aside
  class="fixed md:sticky top-0 left-0 z-50 h-screen bg-black border-r border-zinc-800/90 flex flex-col justify-between transition-[width,transform] duration-200 ease-in-out shrink-0
    {sidebar.isMobileOpen ? 'translate-x-0 w-64' : '-translate-x-full md:translate-x-0'}
    {sidebar.isOpen ? 'md:w-64' : 'md:w-[68px]'}"
>
  <div>
    <!-- Brand Header & Toggle Button -->
    <div class="h-20 flex items-center border-b border-zinc-800/80 transition-all {sidebar.isOpen ? 'justify-between px-5' : 'justify-center px-2'}">
      {#if sidebar.isOpen}
        <a href="/" class="flex flex-col group min-w-0 pr-2">
          <span class="text-xl font-display font-light text-white tracking-[-0.05em] group-hover:text-zinc-200 transition truncate">
            MŌMOS
          </span>
          <span class="text-[9px] text-zinc-500 uppercase tracking-widest font-semibold truncate">
            LEAD SCRAPER
          </span>
        </a>
      {/if}

      <!-- Minimize Toggle (Gemini style) -->
      <button
        onclick={() => sidebar.toggle()}
        class="hidden md:flex items-center justify-center p-2 rounded-xl text-zinc-400 hover:text-white hover:bg-zinc-900 border border-transparent hover:border-zinc-800 transition cursor-pointer shrink-0"
        title={sidebar.isOpen ? 'Collapse sidebar' : 'Expand sidebar'}
        aria-label={sidebar.isOpen ? 'Collapse sidebar' : 'Expand sidebar'}
      >
        {#if sidebar.isOpen}
          <PanelLeftClose class="w-4 h-4" />
        {:else}
          <PanelLeftOpen class="w-4 h-4" />
        {/if}
      </button>

      <!-- Mobile Close -->
      <button
        onclick={() => sidebar.closeMobile()}
        class="md:hidden text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-900"
        aria-label="Close navigation"
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Navigation -->
    <nav class="p-2.5 space-y-1">
      {#each navItems as item}
        {@const isActive = page.url.pathname === item.href || (item.href !== '/' && page.url.pathname.startsWith(item.href))}
        <a
          href={item.href}
          onclick={() => sidebar.closeMobile()}
          title={!sidebar.isOpen ? item.label : undefined}
          class="flex items-center rounded-xl text-xs transition group relative {sidebar.isOpen ? 'gap-3 px-3.5 py-2.5' : 'justify-center p-2.5'} {isActive ? 'bg-white text-black font-semibold shadow-sm' : 'text-zinc-400 hover:text-white hover:bg-zinc-900/80 font-medium'}"
        >
          <item.icon class="w-4 h-4 shrink-0 {isActive ? 'text-black' : 'text-zinc-400 group-hover:text-white'}" />
          {#if sidebar.isOpen}
            <span class="truncate">{item.label}</span>
          {/if}

          <!-- Floating Tooltip when collapsed -->
          {#if !sidebar.isOpen}
            <span class="hidden md:group-hover:block absolute left-full ml-3 px-2.5 py-1 bg-zinc-900 text-white text-xs rounded-lg border border-zinc-700 shadow-xl whitespace-nowrap z-50 pointer-events-none">
              {item.label}
            </span>
          {/if}
        </a>
      {/each}
    </nav>
  </div>

  <!-- Bottom Workspace Status -->
  <div class="p-2.5 border-t border-zinc-900 m-2 rounded-xl bg-zinc-950/60 border border-zinc-800/60 text-xs text-zinc-500">
    {#if sidebar.isOpen}
      <div class="flex items-center justify-between px-1">
        <span>Workspace</span>
        <span class="text-emerald-400 text-xs font-medium flex items-center gap-1.5">
          <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 inline-block animate-pulse"></span>
          Active
        </span>
      </div>
    {:else}
      <div class="flex justify-center" title="Workspace Active">
        <span class="w-2 h-2 rounded-full bg-emerald-400 inline-block animate-pulse"></span>
      </div>
    {/if}
  </div>
</aside>


