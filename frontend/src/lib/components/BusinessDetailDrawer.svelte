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
  <div 
    class="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-xs flex justify-end animate-in fade-in duration-150"
    onclick={onClose}
    role="dialog"
    aria-modal="true"
  >
    <!-- Drawer Panel -->
    <div 
      class="w-full max-w-md bg-slate-900 border-l border-slate-800 h-full p-6 flex flex-col justify-between overflow-y-auto shadow-2xl"
      onclick={(e) => e.stopPropagation()}
    >
      <div>
        <div class="flex items-start justify-between pb-4 border-b border-slate-800">
          <div>
            <Badge variant="purple">{lead.category}</Badge>
            <h2 class="text-lg font-bold text-white mt-2 leading-tight">{lead.name}</h2>
            <div class="flex items-center gap-2 mt-1.5 text-xs text-slate-400">
              <div class="flex items-center gap-1 text-amber-400 font-semibold">
                <Star class="w-3.5 h-3.5 fill-amber-400 text-amber-400" />
                <span>{lead.rating.toFixed(1)}</span>
              </div>
              <span>•</span>
              <span>{lead.review_count} reviews</span>
              {#if lead.price_level}
                <span>•</span>
                <span class="text-slate-300">{lead.price_level}</span>
              {/if}
            </div>
          </div>
          <button onclick={onClose} class="text-slate-400 hover:text-white p-1" aria-label="Close details">
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Details Info -->
        <div class="mt-5 space-y-4 text-xs">
          <!-- Address -->
          <div class="flex items-start gap-3 p-3 rounded-xl bg-slate-950/60 border border-slate-800/80">
            <MapPin class="w-4 h-4 text-indigo-400 shrink-0 mt-0.5" />
            <div>
              <div class="text-slate-400 text-[11px]">Address</div>
              <div class="text-slate-200 mt-0.5">{lead.address || `${lead.city}, ${lead.state}`}</div>
            </div>
          </div>

          <!-- Primary Phone -->
          <div class="flex items-start gap-3 p-3 rounded-xl bg-slate-950/60 border border-slate-800/80">
            <Phone class="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
            <div>
              <div class="text-slate-400 text-[11px]">Primary Phone</div>
              <div class="text-slate-200 font-mono mt-0.5">{lead.primary_phone || 'Not available'}</div>
            </div>
          </div>

          <!-- Website -->
          <div class="flex items-start gap-3 p-3 rounded-xl bg-slate-950/60 border border-slate-800/80">
            <Globe class="w-4 h-4 text-blue-400 shrink-0 mt-0.5" />
            <div class="overflow-hidden">
              <div class="text-slate-400 text-[11px]">Website</div>
              {#if lead.website}
                <a href={lead.website} target="_blank" rel="noreferrer" class="text-indigo-400 hover:underline flex items-center gap-1 mt-0.5 truncate">
                  <span class="truncate">{lead.website}</span>
                  <ExternalLink class="w-3 h-3 shrink-0" />
                </a>
              {:else}
                <div class="text-slate-400 mt-0.5">Not available</div>
              {/if}
            </div>
          </div>

          <!-- Emails -->
          <div class="p-3 rounded-xl bg-slate-950/60 border border-slate-800/80 space-y-1.5">
            <div class="flex items-center gap-2 text-slate-400 text-[11px]">
              <Mail class="w-3.5 h-3.5 text-amber-400" />
              <span>Extracted Emails ({lead.emails?.length || 0})</span>
            </div>
            {#if lead.emails && lead.emails.length > 0}
              <div class="flex flex-wrap gap-1.5 mt-1">
                {#each lead.emails as email}
                  <a href="mailto:{email}" class="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-200 font-mono text-[11px] border border-slate-700">
                    {email}
                  </a>
                {/each}
              </div>
            {:else}
              <div class="text-slate-400 text-[11px]">No direct emails discovered on domain</div>
            {/if}
          </div>

          <!-- Social Channels -->
          <div class="p-3 rounded-xl bg-slate-950/60 border border-slate-800/80 space-y-2">
            <div class="text-slate-400 text-[11px]">Social Media Profiles</div>
            <div class="grid grid-cols-2 gap-2">
              {#if lead.linkedin_url}
                <a href={lead.linkedin_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-[11px]">
                  <Link2 class="w-3.5 h-3.5 text-blue-400" />
                  <span>LinkedIn</span>
                </a>
              {/if}
              {#if lead.facebook_url}
                <a href={lead.facebook_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-[11px]">
                  <Link2 class="w-3.5 h-3.5 text-blue-500" />
                  <span>Facebook</span>
                </a>
              {/if}
              {#if lead.instagram_url}
                <a href={lead.instagram_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-[11px]">
                  <Link2 class="w-3.5 h-3.5 text-pink-400" />
                  <span>Instagram</span>
                </a>
              {/if}
              {#if lead.twitter_url}
                <a href={lead.twitter_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-[11px]">
                  <Link2 class="w-3.5 h-3.5 text-sky-400" />
                  <span>X / Twitter</span>
                </a>
              {/if}
              {#if lead.youtube_url}
                <a href={lead.youtube_url} target="_blank" rel="noreferrer" class="flex items-center gap-2 p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-[11px]">
                  <Link2 class="w-3.5 h-3.5 text-red-400" />
                  <span>YouTube</span>
                </a>
              {/if}
            </div>
            {#if !lead.linkedin_url && !lead.facebook_url && !lead.instagram_url && !lead.twitter_url && !lead.youtube_url}
              <div class="text-slate-400 text-[11px]">No social channels matched</div>
            {/if}
          </div>
        </div>
      </div>

      <div class="pt-4 border-t border-slate-800 flex items-center justify-between">
        {#if lead.google_maps_url}
          <a
            href={lead.google_maps_url}
            target="_blank"
            rel="noreferrer"
            class="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-xs font-medium flex items-center gap-1.5 transition"
          >
            <span>View on Google Maps</span>
            <ExternalLink class="w-3.5 h-3.5" />
          </a>
        {/if}
        <button
          onclick={onClose}
          class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-medium transition"
        >
          Close Drawer
        </button>
      </div>
    </div>
  </div>
{/if}
