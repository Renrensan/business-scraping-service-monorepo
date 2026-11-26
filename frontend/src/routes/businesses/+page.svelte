<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import type { BusinessLead, LeadStatus } from '$lib/types';
  import Badge from '$lib/components/Badge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import BusinessDetailDrawer from '$lib/components/BusinessDetailDrawer.svelte';
  import ExportModal from '$lib/components/ExportModal.svelte';
  import KeywordMultiSelect from '$lib/components/KeywordMultiSelect.svelte';
  import LeadStatusFilter from '$lib/components/LeadStatusFilter.svelte';
  import LeadStatusBadge from '$lib/components/LeadStatusBadge.svelte';
  import NewJobModal from '$lib/components/NewJobModal.svelte';
  import { getWhatsAppUrl } from '$lib/utils/format';
  import {
    Search, Download, Plus, Star,
    Phone, Globe, Mail, RefreshCw, Eye, CheckCircle2, MessageSquare
  } from 'lucide-svelte';

  let leads = $state<BusinessLead[]>([]);
  let totalCount = $state(0);
  let totalPages = $state(1);
  let page = $state(1);
  let pageSize = $state(20);
  let isLoading = $state(true);

  // Dynamically loaded categories and keywords from API
  let availableCategories = $state.raw<string[]>([]);
  let availableKeywords = $state.raw<string[]>([]);

  // Filter states
  let search = $state('');
  let selectedCategory = $state('');
  let selectedKeywords = $state<string[]>([]);
  let selectedStatuses = $state<LeadStatus[]>([]);
  let minRating = $state<string>('');
  let hasEmail = $state(false);
  let hasPhone = $state(false);
  let hasWebsite = $state(false);

  // Drawers & Modals
  let selectedLead = $state<BusinessLead | null>(null);
  let showDrawer = $state(false);
  let showExportModal = $state(false);
  let showNewJobModal = $state(false);

  let searchTimeout: ReturnType<typeof setTimeout> | undefined;

  async function fetchBusinesses() {
    try {
      isLoading = true;
      const params: Record<string, any> = {
        page,
        page_size: pageSize,
        search: search.trim() || undefined,
        category: selectedCategory || undefined,
        keywords: selectedKeywords,
        status: selectedStatuses.length ? selectedStatuses.join(',') : undefined,
        min_rating: minRating ? Number(minRating) : undefined,
        has_email: hasEmail ? true : undefined,
        has_phone: hasPhone ? true : undefined,
        has_website: hasWebsite ? true : undefined
      };

      const res = await api.get<{
        items: BusinessLead[];
        total_count: number;
        total_pages: number;
      }>('/api/v1/businesses', params);

      if (res) {
        leads = res.items || [];
        totalCount = res.total_count || 0;
        totalPages = res.total_pages || 1;
      }
    } catch (err) {
      console.error('Failed to query businesses', err);
    } finally {
      isLoading = false;
    }
  }

  async function toggleContacted(lead: BusinessLead, checked: boolean) {
    const prev = lead.contacted;
    const prevStatus = lead.lead_status;
    lead.contacted = checked;
    lead.lead_status = checked ? (lead.lead_status ?? 'cold') : null;
    try {
      await api.patch(`/api/v1/businesses/${lead.id}/contacted`, { contacted: checked });
      toast.success(checked ? `Marked "${lead.name}" as contacted` : `Unmarked "${lead.name}"`);
    } catch (err) {
      lead.contacted = prev;
      lead.lead_status = prevStatus;
      toast.error('Failed to update contacted status');
    }
  }

  async function setStatus(lead: BusinessLead, status: LeadStatus) {
    const prev = { contacted: lead.contacted, lead_status: lead.lead_status };
    lead.lead_status = status;
    lead.contacted = true;
    try {
      await api.patch(`/api/v1/businesses/${lead.id}/status`, { status });
      toast.success(`"${lead.name}" is now ${status.toUpperCase()}`);
      // Drop it from view if it no longer matches the active status filter
      if (selectedStatuses.length && !selectedStatuses.includes(status)) fetchBusinesses();
    } catch {
      lead.contacted = prev.contacted;
      lead.lead_status = prev.lead_status;
    }
  }

  async function loadCategories() {
    try {
      const res = await api.get<string[]>('/api/v1/businesses/categories');
      if (Array.isArray(res)) {
        availableCategories = res.filter(Boolean);
      }
    } catch {
      // silently ignore
    }
  }

  async function loadKeywords() {
    try {
      const res = await api.get<string[]>('/api/v1/businesses/keywords');
      if (Array.isArray(res)) {
        availableKeywords = res.filter(Boolean);
      }
    } catch {
      // silently ignore
    }
  }

  function handleSearchInput() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      page = 1;
      fetchBusinesses();
    }, 300);
  }

  function openDetail(lead: BusinessLead) {
    selectedLead = lead;
    showDrawer = true;
  }

  onMount(() => {
    fetchBusinesses();
    loadCategories();
    loadKeywords();
  });
</script>

<div class="space-y-5">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-zinc-900">
    <div>
      <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Raw Data</h1>
      <p class="text-xs text-zinc-500 mt-0.5">{totalCount.toLocaleString()} raw entries in database</p>
    </div>
    <div class="flex items-center gap-2.5">
      <button
        onclick={() => (showExportModal = true)}
        class="px-3.5 py-2 rounded-xl border border-zinc-800 bg-zinc-950 text-zinc-300 hover:bg-zinc-900 hover:text-white flex items-center gap-1.5 text-xs font-medium transition cursor-pointer"
      >
        <Download class="w-3.5 h-3.5" />
        <span>Export CSV</span>
      </button>
      <button
        onclick={() => (showNewJobModal = true)}
        class="px-4 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black flex items-center gap-1.5 text-xs font-semibold shadow-sm transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>Scrape</span>
      </button>
    </div>
  </div>

  <!-- Filter Bar -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-4 shadow-sm space-y-3.5">

    <!-- Top Row: Search, Keyword, Category, Rating -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 items-start">
      <!-- Search -->
      <div class="relative">
        <Search class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
        <input
          type="text"
          bind:value={search}
          oninput={handleSearchInput}
          placeholder="Search name, phone, domain..."
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-500 text-xs"
        />
      </div>

      <!-- Scrape Keywords (multi-select) — dynamic from DB -->
      <KeywordMultiSelect
        options={availableKeywords}
        selected={selectedKeywords}
        onChange={(kws) => { selectedKeywords = kws; page = 1; fetchBusinesses(); }}
      />

      <!-- Category — dynamic from DB -->
      <select
        bind:value={selectedCategory}
        onchange={() => { page = 1; fetchBusinesses(); }}
        class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3 py-2 text-zinc-200 focus:outline-hidden focus:border-zinc-500 text-xs cursor-pointer"
      >
        <option value="">All Categories</option>
        {#each availableCategories as cat (cat)}
          <option value={cat}>{cat}</option>
        {/each}
      </select>

      <!-- Rating -->
      <select
        bind:value={minRating}
        onchange={() => { page = 1; fetchBusinesses(); }}
        class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3 py-2 text-zinc-200 focus:outline-hidden focus:border-zinc-500 text-xs cursor-pointer"
      >
        <option value="">All Ratings</option>
        <option value="4.5">★ 4.5 and above</option>
        <option value="4.0">★ 4.0 and above</option>
        <option value="3.5">★ 3.5 and above</option>
      </select>
    </div>

    <!-- Bottom Row: Clean Checkbox Toggles -->
    <div class="flex flex-wrap items-center gap-2 pt-2.5 border-t border-zinc-900/80">
      <span class="text-[11px] font-medium text-zinc-500 uppercase tracking-wider mr-1">Require:</span>

      <!-- Has Email Checkbox -->
      <label class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl border transition cursor-pointer select-none text-xs {hasEmail ? 'bg-white/10 border-white/30 text-white font-medium shadow-xs' : 'bg-zinc-900/60 border-zinc-800/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'}">
        <input
          type="checkbox"
          checked={hasEmail}
          onchange={(e) => {
            hasEmail = e.currentTarget.checked;
            page = 1;
            fetchBusinesses();
          }}
          class="w-3.5 h-3.5 rounded border-zinc-700 bg-zinc-950 text-white accent-white cursor-pointer"
        />
        <Mail class="w-3.5 h-3.5 {hasEmail ? 'text-emerald-400' : 'text-zinc-500'}" />
        <span>Has Email</span>
      </label>

      <!-- Has Phone Checkbox -->
      <label class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl border transition cursor-pointer select-none text-xs {hasPhone ? 'bg-white/10 border-white/30 text-white font-medium shadow-xs' : 'bg-zinc-900/60 border-zinc-800/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'}">
        <input
          type="checkbox"
          checked={hasPhone}
          onchange={(e) => {
            hasPhone = e.currentTarget.checked;
            page = 1;
            fetchBusinesses();
          }}
          class="w-3.5 h-3.5 rounded border-zinc-700 bg-zinc-950 text-white accent-white cursor-pointer"
        />
        <Phone class="w-3.5 h-3.5 {hasPhone ? 'text-emerald-400' : 'text-zinc-500'}" />
        <span>Has Phone</span>
      </label>

      <!-- Has Website Checkbox -->
      <label class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl border transition cursor-pointer select-none text-xs {hasWebsite ? 'bg-white/10 border-white/30 text-white font-medium shadow-xs' : 'bg-zinc-900/60 border-zinc-800/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'}">
        <input
          type="checkbox"
          checked={hasWebsite}
          onchange={(e) => {
            hasWebsite = e.currentTarget.checked;
            page = 1;
            fetchBusinesses();
          }}
          class="w-3.5 h-3.5 rounded border-zinc-700 bg-zinc-950 text-white accent-white cursor-pointer"
        />
        <Globe class="w-3.5 h-3.5 {hasWebsite ? 'text-emerald-400' : 'text-zinc-500'}" />
        <span>Has Website</span>
      </label>

      <div class="hidden sm:block w-px h-5 bg-zinc-800 mx-1"></div>

      <LeadStatusFilter
        selected={selectedStatuses}
        onChange={(sts) => { selectedStatuses = sts; page = 1; fetchBusinesses(); }}
      />
    </div>
  </div>

  <!-- Data Table -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl overflow-hidden shadow-sm">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead class="bg-black border-b border-zinc-800 text-[11px] uppercase tracking-wider text-zinc-400 font-semibold">
          <tr>
            <th class="px-4 py-3.5 w-12 text-center" title="Mark if already contacted">Done</th>
            <th class="px-4 py-3.5">Business Name</th>
            <th class="px-4 py-3.5">Category</th>
            <th class="px-4 py-3.5">Website</th>
            <th class="px-4 py-3.5">Location</th>
            <th class="px-4 py-3.5">Rating</th>
            <th class="px-4 py-3.5">Contacts</th>
            <th class="px-4 py-3.5 text-right">Details</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-900 text-zinc-300">
          {#if isLoading}
            <tr>
              <td colspan="8" class="px-5 py-16 text-center text-zinc-500">
                <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-zinc-400" />
                <span>Loading raw data...</span>
              </td>
            </tr>
          {:else if leads.length === 0}
            <tr>
              <td colspan="8" class="px-5 py-16 text-center text-zinc-500">
                No entries match your filters.
              </td>
            </tr>
          {:else}
            {#each leads as lead (lead.id)}
              <tr class="hover:bg-zinc-900/40 transition {lead.contacted ? 'bg-emerald-950/10 text-zinc-400' : ''}">
                <!-- Contacted Checkbox Column -->
                <td class="px-4 py-3.5 text-center">
                  <input
                    type="checkbox"
                    checked={lead.contacted ?? false}
                    onchange={(e) => toggleContacted(lead, e.currentTarget.checked)}
                    class="w-4 h-4 rounded border-zinc-700 bg-zinc-900 accent-emerald-500 cursor-pointer"
                    title={lead.contacted ? 'Contacted - Click to unmark' : 'Click to mark as contacted'}
                  />
                </td>

                <td class="px-4 py-3.5 font-medium max-w-xs truncate {lead.contacted ? 'text-zinc-400' : 'text-white'}">
                  <div class="flex items-center gap-1.5">
                    <button
                      onclick={() => openDetail(lead)}
                      class="hover:text-zinc-300 text-left truncate font-semibold block transition cursor-pointer"
                    >
                      {lead.name}
                    </button>
                    {#if lead.contacted}
                      <LeadStatusBadge status={lead.lead_status ?? 'cold'} />
                    {/if}
                  </div>
                  <div class="text-[11px] text-zinc-500 font-normal truncate">
                    {lead.street || lead.address}
                  </div>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <Badge variant="default">{lead.category}</Badge>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  {#if lead.website}
                    <a
                      href={lead.website.startsWith('http') ? lead.website : `https://${lead.website}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-emerald-950/70 border border-emerald-800/80 text-emerald-400 text-[11px] font-mono hover:bg-emerald-900/80 transition max-w-[140px] truncate"
                      title={lead.website}
                    >
                      <Globe class="w-3 h-3 shrink-0" />
                      <span class="truncate">{lead.website.replace(/^https?:\/\/(www\.)?/, '').replace(/\/$/, '')}</span>
                    </a>
                  {:else}
                    <span class="text-zinc-600 text-[11px]">—</span>
                  {/if}
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap text-zinc-400">
                  {lead.city || 'N/A'}{lead.state ? `, ${lead.state}` : ''}
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <div class="flex items-center gap-1 font-medium {lead.contacted ? 'text-zinc-300' : 'text-white'}">
                    <Star class="w-3.5 h-3.5 fill-white text-white" />
                    <span>{lead.rating.toFixed(1)}</span>
                    <span class="text-[10px] text-zinc-500">({lead.review_count})</span>
                  </div>
                </td>

                <td class="px-4 py-3.5">
                  <div class="flex items-center gap-1.5">
                    {#if lead.emails && lead.emails.length > 0}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-zinc-900 text-emerald-400 border border-zinc-800 text-[10px] font-mono">
                        <Mail class="w-2.5 h-2.5" />
                        {lead.emails.length}
                      </span>
                    {/if}
                    {#if lead.primary_phone}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-zinc-900 text-zinc-300 border border-zinc-800 text-[10px] font-mono">
                        <Phone class="w-2.5 h-2.5" />
                      </span>
                      {#if getWhatsAppUrl(lead.primary_phone)}
                        <a
                          href={getWhatsAppUrl(lead.primary_phone)}
                          target="_blank"
                          rel="noreferrer"
                          class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-emerald-950/80 hover:bg-emerald-900 text-emerald-400 border border-emerald-800/80 text-[10px] transition"
                          title="Chat on WhatsApp"
                        >
                          <MessageSquare class="w-2.5 h-2.5" />
                        </a>
                      {/if}
                    {/if}
                    {#if !lead.emails?.length && !lead.primary_phone}
                      <span class="text-zinc-600 text-[11px]">—</span>
                    {/if}
                  </div>
                </td>

                <td class="px-4 py-3.5 text-right whitespace-nowrap">
                  <button
                    onclick={() => openDetail(lead)}
                    class="p-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 hover:text-white transition cursor-pointer"
                    title="View details"
                  >
                    <Eye class="w-3.5 h-3.5" />
                  </button>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <Pagination
      page={page}
      totalPages={totalPages}
      onPageChange={(newPage) => {
        page = newPage;
        fetchBusinesses();
      }}
    />
  </div>
</div>

<BusinessDetailDrawer
  lead={selectedLead}
  isOpen={showDrawer}
  onClose={() => (showDrawer = false)}
  onToggleContacted={(l, contacted) => toggleContacted(l, contacted)}
  onStatusChange={(l, status) => setStatus(l, status)}
/>

<ExportModal
  isOpen={showExportModal}
  onClose={() => (showExportModal = false)}
  filterParams={{
    search,
    category: selectedCategory,
    keywords: selectedKeywords,
    status: selectedStatuses.length ? selectedStatuses.join(',') : undefined,
    has_email: hasEmail ? true : undefined,
    has_phone: hasPhone ? true : undefined,
    has_website: hasWebsite ? true : undefined,
    min_rating: minRating
  }}
/>

<NewJobModal
  isOpen={showNewJobModal}
  onClose={() => (showNewJobModal = false)}
  onJobCreated={fetchBusinesses}
/>