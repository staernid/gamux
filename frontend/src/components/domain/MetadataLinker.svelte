<script>
  import { onMount } from 'svelte';
  import { AppAPI } from '../../api/app.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { selectedGame, updateGameInStore, searchStore } from '../../stores/library.js';
  import { isSamePath, normalizeMetadataProvider } from '../../utils/game.js';
  import Icon from '../common/Icon.svelte';

  /**
   * @typedef {Object} Props
   * @property {import('../../types/domain').Game} [game]
   * @property {((game: import('../../types/domain').Game) => void) | null} [onlinked]
   * @property {((game: import('../../types/domain').Game) => void) | null} [onunlinked]
   * @property {((game: import('../../types/domain').Game) => void) | null} [onupdated]
   */

  /** @type {Props} */
  let {
    game = $bindable(null),
    onlinked = null,
    onunlinked = null,
    onupdated = null
  } = $props();

  let targetStore = $state('steam'); // 'steam' | 'gog' | 'rawg' | 'standalone'
  let providers = $state([
    { id: 'steam', displayName: 'Steam', badge: 'AppID' },
    { id: 'gog', displayName: 'GOG.com', badge: 'GOG ID' },
    { id: 'rawg', displayName: 'RAWG Database', badge: 'Open DB' },
    { id: 'standalone', displayName: 'Standalone / DRM-Free', badge: 'Direct' },
  ]);

  let metaQuery = $state('');
  let customId = $state('');
  let metaCandidates = $state([]);
  let isSearchingMeta = $state(false);
  let isLinkingMeta = $state(false);
  let selectedMetaCandidate = $state(null);

  let gameName = $derived(game?.name || '');
  let gameDir = $derived(game?.gameDir || '');
  let appId = $derived(game?.appId || '');

  onMount(() => {
    const providerKey = (game?.metadataProvider || game?.store || 'steam').toLowerCase();
    if (providerKey === 'gog') targetStore = 'gog';
    else if (providerKey === 'rawg') targetStore = 'rawg';
    else if (providerKey === 'standalone' || providerKey === 'custom') targetStore = 'standalone';
    else targetStore = 'steam';

    metaQuery = gameName || '';
    customId = String(game?.metadataId || appId || '');
    if (customId === '0') customId = '';

    AppAPI.listMetadataProviders()
      .then(fetched => {
        if (Array.isArray(fetched) && fetched.length > 0) {
          const list = fetched
            .map(normalizeMetadataProvider)
            .filter(p => p && p.id !== 'steamgriddb')
            .map(p => ({
              id: p.id,
              displayName: p.displayName,
              badge: p.id === 'steam' ? 'AppID' : p.id === 'gog' ? 'GOG ID' : p.id === 'rawg' ? 'Open DB' : 'Direct',
            }));
          list.push({ id: 'standalone', displayName: 'Standalone / DRM-Free', badge: 'Direct' });
          providers = list;
        }
      })
      .catch(() => {});

    if (targetStore !== 'standalone' && metaQuery.trim()) {
      handleSearchMeta();
    }
  });

  export async function handleSearchMeta(storeOverride = null) {
    if (!metaQuery.trim()) return;
    const storeToMatch = (storeOverride || targetStore || 'steam').toLowerCase();
    isSearchingMeta = true;
    selectedMetaCandidate = null;
    try {
      const results = await searchStore(metaQuery.trim());
      metaCandidates = (results || []).filter(c => {
        const storeLower = (c.store || '').toLowerCase();
        return storeLower === storeToMatch;
      });
    } catch {
      metaCandidates = [];
    } finally {
      isSearchingMeta = false;
    }
  }

  function handleSelectMetaCandidate(cand) {
    selectedMetaCandidate = cand;
    customId = cand.id;
  }

  export async function handleApplyMetaLink(candidateToLink = null) {
    if (!gameDir) return;
    if (candidateToLink) {
      handleSelectMetaCandidate(candidateToLink);
    }
    isLinkingMeta = true;
    try {
      const idToLink = targetStore === 'standalone' ? '0' : customId.trim();
      const updated = await AppAPI.linkGameMetadata(gameDir, targetStore, idToLink);
      notificationStore.items.add({
        type: 'success',
        title: 'Metadata Linked',
        message: `Linked ${gameName} to ${targetStore.toUpperCase()} ${idToLink ? `(#${idToLink})` : ''}`,
      });
      if (updated) {
        game = { ...game, ...updated };
        updateGameInStore(updated);
      }
      onupdated?.(updated);
      onlinked?.({ store: targetStore, id: idToLink, game: updated });
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Linking Failed',
        message: err?.message || String(err),
      });
    } finally {
      isLinkingMeta = false;
    }
  }

  export async function handleUnlinkMeta() {
    if (!gameDir) return;
    isLinkingMeta = true;
    try {
      const updated = await AppAPI.unlinkGameMetadata(gameDir);
      notificationStore.items.add({
        type: 'info',
        title: 'Metadata Reset',
        message: `Restored auto-detected identity for ${gameName}.`,
      });
      if (updated) {
        game = { ...game, ...updated };
        updateGameInStore(updated);
      }
      customId = '';
      metaCandidates = [];
      onupdated?.(updated);
      onunlinked?.(updated);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Unlink Failed',
        message: err?.message || String(err),
      });
    } finally {
      isLinkingMeta = false;
    }
  }

  function handleStoreChange(storeId) {
    targetStore = storeId;
    selectedMetaCandidate = null;
    if (targetStore !== 'standalone' && metaQuery.trim()) {
      handleSearchMeta(storeId);
    }
  }
</script>

<div class="space-y-4">
  <!-- Store Provider Selection & Reset Auto -->
  <div class="flex items-center justify-between gap-3">
    <div class="flex items-center gap-2 overflow-x-auto custom-scrollbar pb-1">
      {#each providers as prov}
        {@const isActive = targetStore === prov.id}
        <button
          type="button"
          onclick={() => handleStoreChange(prov.id)}
          class="px-3.5 py-1.5 rounded-xl font-bold text-xs transition-all flex items-center space-x-1.5 flex-shrink-0 cursor-pointer {isActive ? 'bg-white text-neutral-950 shadow-sm' : 'bg-surface-card border border-surface-border text-neutral-400 hover:text-white hover:bg-surface-cardHover'}"
        >
          <span>{prov.displayName}</span>
          <span class="text-3xs px-1.5 py-0.5 rounded font-mono {isActive ? 'bg-neutral-200 text-neutral-800' : 'bg-surface-input border border-surface-border text-neutral-500'}">
            {prov.badge}
          </span>
        </button>
      {/each}
    </div>

    <button
      type="button"
      onclick={handleUnlinkMeta}
      disabled={isLinkingMeta}
      class="px-3 py-1.5 rounded-xl bg-surface-card hover:bg-surface-cardHover border border-surface-border text-neutral-300 hover:text-white text-xs font-bold transition-colors cursor-pointer flex-shrink-0"
      title="Reset metadata overrides to auto-detected origin"
    >
      Reset to Auto
    </button>
  </div>

  {#if targetStore === 'standalone'}
    <!-- Standalone / DRM-Free Configuration Card -->
    <div class="bg-surface-card border border-surface-border p-6 rounded-2xl space-y-4">
      <div class="flex items-start space-x-3.5">
        <div class="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 flex-shrink-0">
          <Icon name="check" className="w-5 h-5" />
        </div>
        <div class="space-y-1">
          <h3 class="text-base font-bold text-white">Standalone / DRM-Free Mode</h3>
          <p class="text-xs text-neutral-400 leading-relaxed">
            Configuring this game as Standalone executes the binary directly via your selected runner without injecting Steamworks emulators or looking up Steam DRM keys.
          </p>
        </div>
      </div>
      <div class="pt-3 border-t border-surface-border flex justify-end">
        <button
          type="button"
          onclick={() => handleApplyMetaLink()}
          disabled={isLinkingMeta}
          class="px-5 py-2 bg-white hover:bg-neutral-200 text-neutral-950 font-bold text-xs rounded-xl transition-all shadow cursor-pointer"
        >
          {isLinkingMeta ? 'Updating...' : 'Set as Standalone'}
        </button>
      </div>
    </div>
  {:else}
    <!-- Store Search & Manual ID Row -->
    <div class="bg-surface-card border border-surface-border p-4 rounded-2xl space-y-3">
      <div class="flex flex-col sm:flex-row items-center gap-3">
        <div class="relative flex-1 w-full">
          <input
            type="text"
            bind:value={metaQuery}
            onkeydown={(e) => { if (e.key === 'Enter') handleSearchMeta(); }}
            placeholder={`Search ${targetStore.toUpperCase()} catalogue...`}
            class="w-full bg-surface-input border border-surface-border focus:border-neutral-400 rounded-xl px-4 py-2 text-xs text-white outline-none font-mono placeholder:text-neutral-500 transition-colors"
          />
        </div>
        <button
          type="button"
          onclick={() => handleSearchMeta()}
          disabled={isSearchingMeta}
          class="w-full sm:w-auto px-5 py-2 bg-surface-card hover:bg-surface-cardHover text-white text-xs font-bold rounded-xl border border-surface-border transition-colors flex-shrink-0 cursor-pointer"
        >
          {isSearchingMeta ? 'Searching...' : 'Search'}
        </button>
      </div>

      <!-- Manual ID Input & Apply Button -->
      <div class="flex items-center gap-3 pt-3 border-t border-surface-border">
        <span class="text-2xs font-mono font-bold text-neutral-400 flex-shrink-0">
          {targetStore === 'steam' ? 'Steam AppID:' : targetStore === 'gog' ? 'GOG ID / Slug:' : 'RAWG Game ID / Slug:'}
        </span>
        <input
          type="text"
          bind:value={customId}
          placeholder={targetStore === 'steam' ? 'e.g. 1091500' : targetStore === 'gog' ? 'e.g. the_witcher_3_wild_hunt' : 'e.g. portal-2 or 4200'}
          class="flex-1 bg-surface-input border border-surface-border focus:border-neutral-400 rounded-lg px-3 py-1 text-xs text-white outline-none font-mono placeholder:text-neutral-600 transition-colors"
        />
        <button
          type="button"
          onclick={() => handleApplyMetaLink()}
          disabled={!customId.trim() || isLinkingMeta}
          class="px-5 py-1.5 bg-white hover:bg-neutral-200 disabled:opacity-40 text-neutral-950 text-xs font-bold rounded-lg transition-colors flex-shrink-0 shadow cursor-pointer"
        >
          {isLinkingMeta ? 'Linking...' : 'Link Game'}
        </button>
      </div>
    </div>

    <!-- Candidate Results List -->
    {#if isSearchingMeta}
      <div class="h-40 flex flex-col items-center justify-center text-neutral-400 space-y-2 font-mono text-xs">
        <Icon name="spinner" className="w-6 h-6 text-white" />
        <p>Querying {targetStore} catalogue...</p>
      </div>
    {:else if metaCandidates.length > 0}
      <div class="space-y-2 pb-6">
        <span class="text-3xs font-mono font-bold uppercase tracking-wider text-neutral-500 block px-1">
          Matching {targetStore} Titles ({metaCandidates.length})
        </span>
        <div class="grid grid-cols-1 gap-2">
          {#each metaCandidates as cand}
            {@const isSelected = selectedMetaCandidate?.id === cand.id || customId === cand.id}
            <div class="flex items-center justify-between p-3 rounded-xl bg-surface-card border transition-all duration-150 {isSelected ? 'border-neutral-300 ring-1 ring-white/20' : 'border-surface-border hover:border-neutral-500'}">
              <div class="flex items-center space-x-3 min-w-0">
                {#if cand.poster_url}
                  <img src={cand.poster_url} alt="" class="w-10 h-14 object-cover rounded-lg bg-black/40 flex-shrink-0 border border-white/5" />
                {/if}
                <div class="min-w-0">
                  <h4 class="text-xs font-bold text-white truncate" title={cand.name}>{cand.name}</h4>
                  <div class="flex items-center space-x-2 text-3xs font-mono text-neutral-400 mt-0.5">
                    <span>ID: <strong class="text-neutral-200">{cand.id}</strong></span>
                    {#if cand.release_date}
                      <span>• Released: {cand.release_date}</span>
                    {/if}
                  </div>
                </div>
              </div>
              <div class="flex items-center space-x-2 flex-shrink-0 pl-3">
                <button
                  type="button"
                  onclick={() => handleApplyMetaLink(cand)}
                  class="px-4 py-1.5 bg-white hover:bg-neutral-200 text-neutral-950 text-xs font-bold rounded-lg transition-colors shadow cursor-pointer"
                >
                  Link Game
                </button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>
