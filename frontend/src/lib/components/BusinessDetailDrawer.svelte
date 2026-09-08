<script lang="ts">
  import type { BusinessLead } from '$lib/types';
  import Badge from '$lib/components/Badge.svelte';
  import { 
    X, MapPin, Phone, Globe, Star, Mail, 
    Link2, ExternalLink 
  } from 'lucide-svelte';

  let {
    lead = null,
    isOpen = false,
    onClose
  }: {
    lead: BusinessLead | null;
    isOpen: boolean;
    onClose: () => void;
  } = $props();
</script>

{#if isOpen && lead}
  <!-- Backdrop -->
  <button 
    type="button"
    aria-label="Close details backdrop"
    class="fixed inset-0 z-50 bg-black/80 backdrop-blur-xs w-full h-full cursor-default"
    onclick={onClose}
  ></button>

  <!-- Drawer Panel -->
  <aside 
    class="fixed top-0 right-0 z-50 w-full max-w-md bg-zinc-950 border-l border-zinc-800 h-full p-6 flex flex-col justify-between overflow-y-auto shadow-2xl font-mono animate-in slide-in-from-right duration-150"
  >
    <div>
      <div class="flex items-start justify-between pb-4 border-b border-zinc-800">
        <div>
          <Badge variant="default">{lead.category}</Badge>
          <h2 class="text-xl font-display font-light text-white mt-2 leading-tight">{lead.name}</h2>
          <div class="flex items-center gap-2 mt-1.5 text-xs text-zinc-500">
            <div class="flex items-center gap-1 text-white font-semibold">
              <Star class="w-3.5 h-3.5 fill-white text-white" />
              <span>{lead.rating.toFixed(1)}</span>
            </div>
            <span>•</span>
            <span>{lead.review_count} reviews</span>
            {#if lead.price_level}
              <span>•</span>
              <span class="text-zinc-300">{lead.price_level}</span>
            {/if}
          </div>
        </div>
        <button onclick={onClose} class="text-zinc-500 hover:text-white p-1 transition cursor-pointer" aria-label="Close details">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Details Info -->
      <div class="mt-5 space-y-3.5 text-xs">
        <!-- Address -->
        <div class="flex items-start gap-3 p-3 rounded-xl bg-black border border-zinc-800">
          <MapPin class="w-4 h-4 text-zinc-400 shrink-0 mt-0.5" />
          <div>
            <div class="text-zinc-500 text-[10px] uppercase tracking-wider">Address</div>
            <div class="text-zinc-200 mt-0.5">{lead.address || `${lead.city}, ${lead.state}`}</div>
          </div>
        </div>

        <!-- Primary Phone -->
        <div class="flex items-start gap-3 p-3 rounded-xl bg-black border border-zinc-800">
          <Phone class="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
          <div>
            <div class="text-zinc-500 text-[10px] uppercase tracking-wider">Primary Phone</div>
            <div class="text-white font-mono mt-0.5">{lead.primary_phone || 'Not available'}</div>
          </div>
        </div>

        <!-- Website -->
        <div class="flex items-start gap-3 p-3 rounded-xl bg-black border border-zinc-800">
          <Globe class="w-4 h-4 text-zinc-400 shrink-0 mt-0.5" />
          <div class="overflow-hidden">
            <div class="text-zinc-500 text-[10px] uppercase tracking-wider">Website URL</div>
            {#if lead.website}
              <a href={lead.website} target="_blank" rel="noreferrer" class="text-zinc-200 hover:text-white hover:underline flex items-center gap-1 mt-0.5 truncate">
                <span class="truncate">{lead.website}</span>
                <ExternalLink class="w-3 h-3 shrink-0" />
              </a>
            {:else}
              <div class="text-zinc-500 mt-0.5">Not available</div>
            {/if}
          </div>
        </div>

        <!-- Emails -->
        <div class="p-3.5 rounded-xl bg-black border border-zinc-800 space-y-1.5">
          <div class="flex items-center gap-2 text-zinc-500 text-[10px] uppercase tracking-wider">
            <Mail class="w-3.5 h-3.5 text-emerald-400" />
            <span>Extracted Emails ({lead.emails?.length || 0})</span>
          </div>
          {#if lead.emails && lead.emails.length > 0}
            <div class="flex flex-wrap gap-1.5 mt-1">
              {#each lead.emails as email}
                <a href="mailto:{email}" class="px-2 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-emerald-400 font-mono text-[11px] border border-zinc-800">
                  {email}
                </a>
              {/each}
            </div>
          {:else}
            <div class="text-zinc-600 text-[11px]">No direct emails discovered on domain</div>
          {/if}
        </div>

        <!-- Social Channels -->
        <div class="p-3.5 rounded-xl bg-black border border-zinc-800 space-y-2">
          <div class="text-zinc-500 text-[10px] uppercase tracking-wider">Social Media Handles</div>
          <div class="grid grid-cols-2 gap-2">
            {#if lead.linkedin_url}
              <a href={lead.linkedin_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-800 text-[11px]">
                <Link2 class="w-3.5 h-3.5 text-zinc-400" />
                <span>LinkedIn</span>
              </a>
            {/if}
            {#if lead.facebook_url}
              <a href={lead.facebook_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-800 text-[11px]">
                <Link2 class="w-3.5 h-3.5 text-zinc-400" />
                <span>Facebook</span>
              </a>
            {/if}
            {#if lead.instagram_url}
              <a href={lead.instagram_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-800 text-[11px]">
                <Link2 class="w-3.5 h-3.5 text-zinc-400" />
                <span>Instagram</span>
              </a>
            {/if}
            {#if lead.twitter_url}
              <a href={lead.twitter_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-800 text-[11px]">
                <Link2 class="w-3.5 h-3.5 text-zinc-400" />
                <span>X / Twitter</span>
              </a>
            {/if}
            {#if lead.youtube_url}
              <a href={lead.youtube_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-800 text-[11px]">
                <Link2 class="w-3.5 h-3.5 text-zinc-400" />
                <span>YouTube</span>
              </a>
            {/if}
          </div>
          {#if !lead.linkedin_url && !lead.facebook_url && !lead.instagram_url && !lead.twitter_url && !lead.youtube_url}
            <div class="text-zinc-600 text-[11px]">No social channels matched</div>
          {/if}
        </div>
      </div>
    </div>

    <div class="pt-4 border-t border-zinc-800 flex items-center justify-between">
      {#if lead.google_maps_url}
        <a
          href={lead.google_maps_url}
          target="_blank"
          rel="noreferrer"
          class="px-4 py-2 rounded-xl bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-800 text-xs font-mono flex items-center gap-1.5 transition"
        >
          <span>Google Maps</span>
          <ExternalLink class="w-3 h-3" />
        </a>
      {/if}
      <button
        onclick={onClose}
        class="px-4 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black text-xs font-semibold font-mono transition cursor-pointer"
      >
        Close
      </button>
    </div>
  </aside>
{/if}

