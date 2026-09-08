<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { BusinessLead } from '$lib/types';
  import { CATEGORIES } from '$lib/constants';
  import Badge from '$lib/components/Badge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import BusinessDetailDrawer from '$lib/components/BusinessDetailDrawer.svelte';
  import ExportModal from '$lib/components/ExportModal.svelte';
  import NewJobModal from '$lib/components/NewJobModal.svelte';
  import { 
    Search, Filter, Download, Plus, Star, MapPin, 
    Phone, Globe, Mail, RefreshCw, Eye
  } from 'lucide-svelte';

  let leads = $state<BusinessLead[]>([]);
  let totalCount = $state(0);
  let totalPages = $state(1);
  let page = $state(1);
  let pageSize = $state(20);
  let isLoading = $state(true);

  // Filter States
  let search = $state('');
  let selectedCategory = $state('');
  let city = $state('');
  let minRating = $state<string>('');
  let hasEmail = $state<boolean | undefined>(undefined);
  let hasPhone = $state<boolean | undefined>(undefined);

  // Drawers & Modals
  let selectedLead = $state<BusinessLead | null>(null);
  let showDrawer = $state(false);
  let showExportModal = $state(false);
  let showNewJobModal = $state(false);

  async function fetchBusinesses() {
    try {
      isLoading = true;
      const params: Record<string, any> = {
        page,
        page_size: pageSize,
        search: search.trim() || undefined,
        category: (selectedCategory && selectedCategory !== 'All Categories') ? selectedCategory : undefined,
        city: city.trim() || undefined,
        min_rating: minRating ? Number(minRating) : undefined,
        has_email: hasEmail !== undefined ? hasEmail : undefined,
        has_phone: hasPhone !== undefined ? hasPhone : undefined
      };

      const res = await api.get<{
        items: BusinessLead[];
        total_count: number;
        total_pages: number;
        page: number;
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

  function handleSearchInput() {
    page = 1;
    fetchBusinesses();
  }

  function openDetail(lead: BusinessLead) {
    selectedLead = lead;
    showDrawer = true;
  }

  onMount(() => {
    fetchBusinesses();
  });
</script>

<div class="space-y-5">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h1 class="text-xl font-bold text-white tracking-tight">Leads Explorer</h1>
      <p class="text-xs text-slate-400 mt-1">Search, filter, and inspect verified Google Maps business listings ({totalCount.toLocaleString()} leads)</p>
    </div>
    <div class="flex items-center gap-2.5">
      <button
        onclick={() => (showExportModal = true)}
        class="px-3.5 py-2 rounded-xl border border-slate-700 bg-slate-800 text-slate-300 hover:bg-slate-700 flex items-center gap-1.5 text-xs font-medium transition cursor-pointer"
      >
        <Download class="w-3.5 h-3.5" />
        <span>Export CSV</span>
      </button>
      <button
        onclick={() => (showNewJobModal = true)}
        class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white flex items-center gap-1.5 text-xs font-medium shadow-md shadow-indigo-600/20 transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>New Extraction</span>
      </button>
    </div>
  </div>

  <!-- Filter Bar -->
  <div class="bg-slate-900 border border-slate-800 rounded-2xl p-4 shadow-sm space-y-3">
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
      <!-- Search Input -->
      <div class="relative">
        <Search class="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
        <input
          type="text"
          bind:value={search}
          oninput={handleSearchInput}
          placeholder="Search business name, phone, url..."
          class="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
        />
      </div>

      <!-- Category Filter -->
      <select
        bind:value={selectedCategory}
        onchange={handleSearchInput}
        class="w-full bg-slate-800/80 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 focus:outline-hidden focus:border-indigo-500 text-xs"
      >
        {#each CATEGORIES as cat}
          <option value={cat}>{cat}</option>
        {/each}
      </select>

      <!-- City Input -->
      <div class="relative">
        <MapPin class="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
        <input
          type="text"
          bind:value={city}
          oninput={handleSearchInput}
          placeholder="Filter by city..."
          class="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
        />
      </div>

      <!-- Rating Filter -->
      <select
        bind:value={minRating}
        onchange={handleSearchInput}
        class="w-full bg-slate-800/80 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 focus:outline-hidden focus:border-indigo-500 text-xs"
      >
        <option value="">All Ratings</option>
        <option value="4.5">★ 4.5 and above</option>
        <option value="4.0">★ 4.0 and above</option>
        <option value="3.5">★ 3.5 and above</option>
      </select>
    </div>

    <!-- Contact Checkbox Filters -->
    <div class="flex flex-wrap items-center gap-4 pt-2 border-t border-slate-800/80 text-xs text-slate-400">
      <label class="flex items-center gap-1.5 cursor-pointer">
        <input
          type="checkbox"
          checked={hasEmail === true}
          onchange={(e) => {
            hasEmail = e.currentTarget.checked ? true : undefined;
            handleSearchInput();
          }}
          class="rounded accent-indigo-600 bg-slate-800"
        />
        <span>Has Verified Email</span>
      </label>

      <label class="flex items-center gap-1.5 cursor-pointer">
        <input
          type="checkbox"
          checked={hasPhone === true}
          onchange={(e) => {
            hasPhone = e.currentTarget.checked ? true : undefined;
            handleSearchInput();
          }}
          class="rounded accent-indigo-600 bg-slate-800"
        />
        <span>Has Primary Phone</span>
      </label>

      <div class="ml-auto text-[11px] text-slate-400">
        Showing {leads.length} of {totalCount} records
      </div>
    </div>
  </div>

  <!-- Data Table -->
  <div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead class="bg-slate-950/60 border-b border-slate-800 text-[11px] uppercase tracking-wider text-slate-400 font-semibold">
          <tr>
            <th class="px-5 py-3.5">Business Name</th>
            <th class="px-4 py-3.5">Category</th>
            <th class="px-4 py-3.5">Location</th>
            <th class="px-4 py-3.5">Rating</th>
            <th class="px-4 py-3.5">Contact & Socials</th>
            <th class="px-4 py-3.5 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60 text-slate-300">
          {#if isLoading}
            <tr>
              <td colspan="6" class="px-5 py-16 text-center text-slate-400">
                <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-indigo-400" />
                <span>Loading leads from database...</span>
              </td>
            </tr>
          {:else if leads.length === 0}
            <tr>
              <td colspan="6" class="px-5 py-16 text-center text-slate-400">
                No business listings match your query. Try broadening your filters!
              </td>
            </tr>
          {:else}
            {#each leads as lead}
              <tr class="hover:bg-slate-800/40 transition">
                <td class="px-5 py-3.5 font-medium text-white max-w-xs truncate">
                  <button 
                    onclick={() => openDetail(lead)}
                    class="hover:text-indigo-400 text-left truncate font-semibold block transition cursor-pointer"
                  >
                    {lead.name}
                  </button>
                  <div class="text-[11px] text-slate-400 font-normal truncate">
                    {lead.street || lead.address}
                  </div>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <Badge variant="purple">{lead.category}</Badge>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap text-slate-400">
                  {lead.city || 'N/A'}{lead.state ? `, ${lead.state}` : ''}
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <div class="flex items-center gap-1 text-amber-400 font-medium">
                    <Star class="w-3.5 h-3.5 fill-amber-400" />
                    <span>{lead.rating.toFixed(1)}</span>
                    <span class="text-[10px] text-slate-400">({lead.review_count})</span>
                  </div>
                </td>

                <td class="px-4 py-3.5">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    {#if lead.emails && lead.emails.length > 0}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[10px] font-mono">
                        <Mail class="w-2.5 h-2.5" />
                        {lead.emails.length}
                      </span>
                    {/if}
                    {#if lead.primary_phone}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20 text-[10px] font-mono">
                        <Phone class="w-2.5 h-2.5" />
                      </span>
                    {/if}
                    {#if lead.website}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700 text-[10px]">
                        <Globe class="w-2.5 h-2.5" />
                      </span>
                    {/if}
                  </div>
                </td>

                <td class="px-4 py-3.5 text-right whitespace-nowrap">
                  <button
                    onclick={() => openDetail(lead)}
                    class="p-1.5 rounded-lg border border-slate-700 bg-slate-800 hover:bg-slate-700 text-slate-300 transition cursor-pointer"
                    title="View Profile Details"
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

    <!-- Pagination -->
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
/>

<ExportModal
  isOpen={showExportModal}
  onClose={() => (showExportModal = false)}
  filterParams={{
    search,
    category: selectedCategory,
    city,
    has_email: hasEmail,
    min_rating: minRating
  }}
/>

<NewJobModal
  isOpen={showNewJobModal}
  onClose={() => (showNewJobModal = false)}
  onJobCreated={fetchBusinesses}
/>