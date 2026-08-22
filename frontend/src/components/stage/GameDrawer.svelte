<script>
  import { selectedGame, showInspector, queueApplyGame } from '../../stores/library.js';
  import { taskQueue } from '../../stores/queue.js';
  import { notificationStore } from '../../stores/notifications.js';
  import { formatNewsHTML } from '../../utils/format.js';
  import { isSteamApp } from '../../utils/game.js';
  import { AppAPI } from '../../api/app.js';

  let activeTab = 'config'; // 'config' | 'integrity' | 'news'
  let integrityReport = null;
  let isCheckingIntegrity = false;
  let newsItems = [];
  let isLoadingNews = false;
  let expandedNewsIndex = 0; // First patch note expanded by default

  // Local form state
  let customExe = '';
  let customArgs = '';
  let customAppID = '';
  let customMode = 'loader';
  let customWinePrefix = '';
  let optAddLutris = true;
  let optNormalize = true;
  let optSteamless = true;
  let optAchievements = true;

  $: candidates = ($selectedGame?.launch_candidates || $selectedGame?.LaunchCandidates || [])
    .filter(c => {
      const exe = (c.executable || c.Executable || '').toLowerCase();
      return !exe.includes('steamclient_loader') && !exe.includes('coldclientloader');
    })
    .map(c => ({
      name: c.name || c.Name || c.executable || c.Executable || '',
      executable: c.executable || c.Executable || '',
      arguments: c.arguments || c.Arguments || '',
      description: c.description || c.Description || '',
    }));

  $: if ($selectedGame) {
    let baseExe = $selectedGame.ExePath || (candidates[0]?.executable || '');
    if (baseExe.toLowerCase().includes('steamclient_loader') || baseExe.toLowerCase().includes('coldclientloader')) {
      baseExe = candidates[0]?.executable || '';
    }
    customExe = baseExe;
    customAppID = $selectedGame.AppID || '';
    customMode = $selectedGame.isPortable ? 'portable' : 'loader';
    customArgs = candidates[0]?.arguments || $selectedGame.ExeArgs || '';
    customWinePrefix = '';
    optAddLutris = true;
    optNormalize = true;
    optSteamless = true;
    optAchievements = true;
    integrityReport = null;
    newsItems = [];
    expandedNewsIndex = 0;
  }

  function handleSelectCandidate(e) {
    const val = e.target.value;
    if (val === '__custom__') return;
    const found = candidates.find(c => c.executable === val);
    if (found) {
      customExe = found.executable;
      if (found.arguments) customArgs = found.arguments;
    } else if (val) {
      customExe = val;
    }
  }

  async function browseExecutable() {
    if (!$selectedGame?.GameDir) return;
    try {
      const selected = await AppAPI.selectFile('Select Game Executable', 'Executables', '*');
      if (selected) {
        let rel = selected;
        if (selected.startsWith($selectedGame.GameDir)) {
          rel = selected.slice($selectedGame.GameDir.length).replace(/^[/\\]+/, '');
        }
        customExe = rel;
      }
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'File Selection Failed',
        message: err?.message || String(err)
      });
    }
  }

  function handleApply() {
    if (!$selectedGame) return;
    const opts = {
      Path: $selectedGame.GameDir,
      ExePath: customExe.trim(),
      ExeArgs: customArgs.trim(),
      AppID: customAppID.trim(),
      Portable: customMode === 'portable',
      ApplyGBE: true,
      AddLutris: optAddLutris,
      NormalizeDir: optNormalize,
      NoSteamless: !optSteamless,
      FetchAchievements: optAchievements,
      WinePrefix: customWinePrefix.trim(),
    };
    queueApplyGame($selectedGame, opts);
    $showInspector = false;
  }

  async function runIntegrityCheck() {
    if (!$selectedGame?.GameDir) return;
    isCheckingIntegrity = true;
    try {
      integrityReport = await AppAPI.verifyIntegrity($selectedGame.GameDir);
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Integrity Check Failed',
        message: err?.message || String(err)
      });
    } finally {
      isCheckingIntegrity = false;
    }
  }

  async function loadNews() {
    if (!$selectedGame?.AppID || $selectedGame.AppID === '0') return;
    isLoadingNews = true;
    try {
      const items = await AppAPI.fetchNews($selectedGame.AppID, 5);
      newsItems = items || [];
    } catch (err) {
      notificationStore.items.add({
        type: 'error',
        title: 'Failed to Fetch News',
        message: err?.message || String(err)
      });
    } finally {
      isLoadingNews = false;
    }
  }

  function toggleNewsExpand(idx) {
    expandedNewsIndex = expandedNewsIndex === idx ? -1 : idx;
  }

  function handleRollback() {
    if (!$selectedGame?.GameDir) return;
    if (confirm(`Rollback emulator modifications for ${$selectedGame.Name}?`)) {
      taskQueue.enqueue({
        type: 'rollback',
        title: `Rollback: ${$selectedGame.Name}`,
        gameName: $selectedGame.Name,
        gameDir: $selectedGame.GameDir,
        run: async () => {
          await AppAPI.rollback($selectedGame.GameDir, false);
          return `Rolled back ${$selectedGame.Name}`;
        },
        onSuccess: async () => {
          const raw = await AppAPI.inspectGameEx($selectedGame.GameDir, 0);
          if (raw) selectedGame.set(raw);
        }
      });
      $showInspector = false;
    }
  }

  $: if ($showInspector && activeTab === 'news' && newsItems.length === 0 && $selectedGame?.AppID) {
    loadNews();
  }
</script>

{#if $showInspector && $selectedGame}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 bg-black/75 backdrop-blur-sm z-40 flex items-center justify-center p-4 sm:p-6 animate-fade-in"
    on:click|self={() => { $showInspector = false; }}
    on:keydown={(e) => { if (e.key === 'Escape') $showInspector = false; }}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <!-- Modal Card -->
    <div class="bg-[#121218] border border-[#2c2c3e] w-full max-w-2xl rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh] animate-slide-up">
      <!-- Modal Header -->
      <div class="px-5 py-3.5 border-b border-[#20202c] flex items-center justify-between bg-[#15151e]">
        <div class="flex items-center space-x-2.5 min-w-0">
          <svg class="w-4 h-4 text-slate-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          <h3 class="text-sm font-bold text-white truncate">{$selectedGame.Name}</h3>
        </div>

        <button
          type="button"
          on:click={() => { $showInspector = false; }}
          class="w-6 h-6 rounded-lg bg-[#1e1e28] hover:bg-[#282836] text-slate-400 hover:text-white flex items-center justify-center text-xs font-bold transition-colors"
        >
          ✕
        </button>
      </div>

      <!-- Navigation Tabs -->
      <div class="px-5 border-b border-[#20202c] bg-[#121218] flex space-x-5 text-xs font-bold">
        <button
          type="button"
          on:click={() => { activeTab = 'config'; }}
          class="py-2.5 transition-colors {activeTab === 'config' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
        >
          Engine
        </button>
        <button
          type="button"
          on:click={() => { activeTab = 'integrity'; }}
          class="py-2.5 transition-colors {activeTab === 'integrity' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
        >
          Integrity
        </button>
        {#if isSteamApp($selectedGame.AppID)}
          <button
            type="button"
            on:click={() => { activeTab = 'news'; }}
            class="py-2.5 transition-colors {activeTab === 'news' ? 'text-white border-b-2 border-white font-black' : 'text-slate-400 hover:text-slate-200'}"
          >
            Patch Notes
          </button>
        {/if}
      </div>

      <!-- Modal Body (Scrollable) -->
      <div class="p-5 overflow-y-auto space-y-3.5 text-xs">
        {#if activeTab === 'config'}
          <!-- Executable & Args -->
          <div class="space-y-3">
            <div>
              <label for="drawerMainExe" class="block text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-1">
                Executable
              </label>
              <div class="space-y-1.5">
                <div class="flex space-x-2">
                  <select
                    id="drawerMainExe"
                    on:change={handleSelectCandidate}
                    value={candidates.some(c => c.executable === customExe) ? customExe : '__custom__'}
                    class="flex-1 bg-[#181822] border border-[#282838] rounded-xl px-3 py-1.5 text-slate-100 outline-none font-mono text-xs focus:border-white"
                  >
                    {#if candidates.length > 0}
                      {#each candidates as cand}
                        <option value={cand.executable}>
                          {cand.name ? `${cand.name} (${cand.executable})` : cand.executable}
                        </option>
                      {/each}
                    {:else if $selectedGame.ExePath}
                      <option value={$selectedGame.ExePath}>{$selectedGame.ExePath} (Auto-detected)</option>
                    {/if}
                    <option value="__custom__">Custom Executable...</option>
                  </select>

                  <button
                    type="button"
                    on:click={browseExecutable}
                    class="bg-[#20202c] hover:bg-[#2a2a3a] text-slate-200 px-3 py-1.5 rounded-xl font-bold border border-[#323244] flex-shrink-0"
                  >
                    Browse
                  </button>
                </div>

                <input
                  type="text"
                  bind:value={customExe}
                  placeholder="Relative path (e.g. Binaries/Win64/Game.exe)"
                  class="w-full bg-[#0d0d12] border border-[#242432] rounded-xl px-3 py-1 text-slate-200 font-mono outline-none focus:border-white text-[11px]"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label for="drawerArgs" class="block text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-1">
                  Launch Flags
                </label>
                <input
                  id="drawerArgs"
                  type="text"
                  bind:value={customArgs}
                  placeholder="-dx11 -fullscreen"
                  class="w-full bg-[#181822] border border-[#282838] rounded-xl px-3 py-1.5 text-slate-100 outline-none font-mono focus:border-white"
                />
              </div>
              <div>
                <label for="drawerMode" class="block text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-1">
                  EMU Mode
                </label>
                <select
                  id="drawerMode"
                  bind:value={customMode}
                  class="w-full bg-[#181822] border border-[#282838] rounded-xl px-3 py-1.5 text-slate-100 outline-none focus:border-white"
                >
                  <option value="loader">Loader (ColdClientLoader)</option>
                  <option value="portable">Portable (Direct DLL)</option>
                </select>
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label for="drawerAppId" class="block text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-1">
                  AppID Override
                </label>
                <input
                  id="drawerAppId"
                  type="text"
                  bind:value={customAppID}
                  placeholder="123456"
                  class="w-full bg-[#181822] border border-[#282838] rounded-xl px-3 py-1.5 text-slate-100 outline-none font-mono focus:border-white"
                />
              </div>
              <div>
                <label for="drawerPrefix" class="block text-slate-400 font-bold uppercase tracking-wider text-[10px] mb-1">
                  Wine Prefix
                </label>
                <input
                  id="drawerPrefix"
                  type="text"
                  bind:value={customWinePrefix}
                  placeholder="~/.wine or custom prefix"
                  class="w-full bg-[#181822] border border-[#282838] rounded-xl px-3 py-1.5 text-slate-100 outline-none font-mono focus:border-white"
                />
              </div>
            </div>

            <!-- Toggles -->
            <div class="grid grid-cols-2 gap-2 pt-2 border-t border-[#20202c]">
              <label class="flex items-center space-x-2 text-slate-300 cursor-pointer select-none">
                <input type="checkbox" bind:checked={optAddLutris} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
                <span>Lutris Registration</span>
              </label>
              <label class="flex items-center space-x-2 text-slate-300 cursor-pointer select-none">
                <input type="checkbox" bind:checked={optNormalize} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
                <span>Normalize Folder</span>
              </label>
              <label class="flex items-center space-x-2 text-slate-300 cursor-pointer select-none">
                <input type="checkbox" bind:checked={optSteamless} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
                <span>Unpack SteamStub</span>
              </label>
              <label class="flex items-center space-x-2 text-slate-300 cursor-pointer select-none">
                <input type="checkbox" bind:checked={optAchievements} class="w-3.5 h-3.5 rounded bg-black border-slate-700" />
                <span>Achievements & DLCs</span>
              </label>
            </div>
          </div>

        {:else if activeTab === 'integrity'}
          <div class="space-y-3">
            <p class="text-slate-400">
              Verify local depot checksums against official SteamPipe SHA-256 manifests:
            </p>

            <button
              type="button"
              on:click={runIntegrityCheck}
              disabled={isCheckingIntegrity}
              class="bg-white hover:bg-neutral-200 text-black font-black px-4 py-2 rounded-xl transition-colors disabled:opacity-50"
            >
              {isCheckingIntegrity ? 'Verifying...' : 'Run Integrity Check'}
            </button>

            {#if integrityReport}
              <div class="bg-[#181822] border border-[#282838] p-3.5 rounded-xl space-y-1 font-mono text-xs">
                <div>Checked: <strong class="text-white">{integrityReport.TotalCount || 0} files</strong></div>
                <div class="text-emerald-400">✓ Valid: {integrityReport.ValidCount || 0}</div>
                {#if integrityReport.CorruptedCount > 0}
                  <div class="text-rose-400">✗ Corrupted: {integrityReport.CorruptedCount}</div>
                {/if}
                {#if integrityReport.MissingCount > 0}
                  <div class="text-amber-400">⚠ Missing: {integrityReport.MissingCount}</div>
                {/if}
              </div>
            {/if}
          </div>

        {:else if activeTab === 'news'}
          <div class="space-y-3">
            {#if isLoadingNews}
              <div class="py-8 text-center text-slate-400 font-mono">Fetching Steam patch notes...</div>
            {:else if newsItems.length > 0}
              {#each newsItems as item, idx}
                {@const isExpanded = expandedNewsIndex === idx}
                <div class="bg-[#151520] border border-[#262638] rounded-2xl overflow-hidden shadow-md transition-all">
                  <!-- Note Header (Click to expand/collapse) -->
                  <div
                    class="p-4 cursor-pointer hover:bg-[#1c1c2a] transition-colors flex items-center justify-between gap-3 select-none"
                    on:click={() => toggleNewsExpand(idx)}
                    role="button"
                    tabindex="0"
                    on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') toggleNewsExpand(idx); }}
                  >
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center space-x-2">
                        <strong class="text-white font-bold text-xs sm:text-sm truncate leading-snug">{item.title}</strong>
                        {#if item.feed_label}
                          <span class="text-[10px] font-mono text-slate-400 bg-black/40 px-1.5 py-0.5 rounded border border-[#2a2a3a]">
                            {item.feed_label}
                          </span>
                        {/if}
                      </div>
                      <div class="text-[11px] text-slate-400 font-mono mt-0.5 flex items-center space-x-2">
                        <span>{new Date(item.date * 1000).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })}</span>
                        {#if item.author}
                          <span>· by {item.author}</span>
                        {/if}
                      </div>
                    </div>

                    <div class="flex items-center space-x-2 flex-shrink-0">
                      {#if item.url}
                        <a
                          href={item.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          on:click|stopPropagation
                          class="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-black/40 text-xs"
                          title="Open on Steam Web"
                        >
                          ↗
                        </a>
                      {/if}
                      <span class="text-slate-400 font-mono text-xs transform transition-transform {isExpanded ? 'rotate-180' : ''}">
                        ▼
                      </span>
                    </div>
                  </div>

                  <!-- Full Rich Formatted Note Content -->
                  {#if isExpanded}
                    <div class="p-4 border-t border-[#202030] bg-[#101017] text-slate-200 leading-relaxed text-xs space-y-2 max-h-96 overflow-y-auto">
                      {@html formatNewsHTML(item.contents)}
                    </div>
                  {/if}
                </div>
              {/each}
            {:else}
              <div class="py-8 text-center text-slate-500 font-mono">No patch notes available for this title.</div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Modal Footer -->
      <div class="px-5 py-3 border-t border-[#20202c] bg-[#15151e] flex items-center justify-between">
        <div>
          {#if $selectedGame.isPatched}
            <button
              type="button"
              on:click={handleRollback}
              class="text-rose-400 hover:text-rose-300 font-bold text-xs"
            >
              Rollback
            </button>
          {/if}
        </div>

        <div class="flex items-center space-x-2">
          <button
            type="button"
            on:click={() => { $showInspector = false; }}
            class="text-slate-400 hover:text-white px-3 py-1.5 font-bold"
          >
            Cancel
          </button>
          <button
            type="button"
            on:click={handleApply}
            class="bg-white hover:bg-neutral-200 active:bg-neutral-300 text-black font-black px-5 py-1.5 rounded-xl transition-colors shadow-sm"
          >
            Apply
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
