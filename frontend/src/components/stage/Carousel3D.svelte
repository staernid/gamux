<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import PosterArt from '../common/PosterArt.svelte';
  import { filteredGames, selectedGame } from '../../stores/library.js';
  import { getPlatformColors, isSamePath } from '../../utils/game.js';

  const dispatch = createEventDispatcher();

  let containerEl;
  let animFrameId = null;

  // High-speed smooth target-following physics
  let currentPos = 0;
  let targetPos = 0;
  let isDragging = false;
  let dragStartX = 0;
  let dragStartPos = 0;
  let lastDragTime = 0;
  let dragVelocity = 0;

  // Wheel input acceleration & thresholding
  let wheelAccumulator = 0;
  let lastWheelTime = 0;
  let wheelResetTimeout = null;

  // Store synchronization guard to prevent circular reactive feedback
  let lastEmittedDir = null;
  let lastSyncIndex = -1;

  $: list = $filteredGames || [];
  $: count = list.length;

  // Synchronize when selectedGame changes externally (e.g. keyboard navigation, search grid, drawer)
  $: if ($selectedGame && count > 0) {
    const dir = $selectedGame.gameDir;
    if (dir !== lastEmittedDir) {
      lastEmittedDir = dir;
      const idx = list.findIndex(g => isSamePath(g.gameDir, dir));
      if (idx >= 0 && !isDragging) {
        const currWrapped = wrapIndex(Math.round(currentPos));
        let diff = idx - currWrapped;
        if (diff > count / 2) diff -= count;
        if (diff < -count / 2) diff += count;
        targetPos = Math.round(currentPos) + diff;
      }
    }
  }

  function wrapIndex(i) {
    if (count === 0) return 0;
    return ((i % count) + count) % count;
  }

  function getGameDiff(index, currPos, total) {
    if (total <= 0) return 0;
    const wrappedPos = ((currPos % total) + total) % total;
    let d = index - wrappedPos;
    if (d > total / 2) d -= total;
    if (d < -total / 2) d += total;
    return d;
  }

  // Smooth continuous 3D card layout along a spatial depth arc (Pure GPU properties only)
  function getCardTransform(diff) {
    const absDiff = Math.abs(diff);

    // Exact center settled card (clean 2D positioning to guarantee WebKit never clips near-plane)
    if (absDiff < 0.0001) {
      return {
        transform: 'translate(-50%, -50%) translate3d(0px, 0px, 0px) scale(1)',
        zIndex: 100,
        opacity: 1,
        dimOpacity: 0,
        centerProximity: 1,
      };
    }

    const sign = diff > 0 ? 1 : -1;

    // Curved horizontal spacing (290px between cards)
    let x = 0;
    if (absDiff <= 1) {
      x = sign * absDiff * 290;
    } else {
      x = sign * (290 + (absDiff - 1) * 260);
    }

    // Depth curve in Z (parabolic arc)
    const z = -Math.pow(absDiff, 1.2) * 80;

    // Y-axis rotation (angled towards center)
    const rotY = -sign * Math.min(42, Math.pow(absDiff, 0.85) * 24);

    // Scale falloff
    const scale = Math.max(0.65, 1 - absDiff * 0.12);

    // Opacity and black dimmer for depth falloff
    const opacity = Math.max(0, 1 - Math.pow(absDiff / 2.6, 1.6));
    const dimOpacity = Math.min(0.65, absDiff * 0.3);

    return {
      transform: `translate(-50%, -50%) translate3d(${x.toFixed(1)}px, 0px, ${z.toFixed(1)}px) rotateY(${rotY.toFixed(1)}deg) scale(${scale.toFixed(3)})`,
      zIndex: Math.round(100 - absDiff * 20),
      opacity: Math.max(0, Math.min(1, opacity)),
      dimOpacity: dimOpacity,
      centerProximity: Math.max(0, 1 - absDiff * 1.5),
    };
  }

  // Render 5 visible active games within view window with stable DOM identity
  $: visibleCards = count > 0
    ? list.map((game, index) => {
        const diff = getGameDiff(index, currentPos, count);
        const isVisible = count <= 5 || Math.abs(diff) <= 2.6;
        return {
          game,
          index,
          diff,
          isVisible,
          style: isVisible ? getCardTransform(diff) : null,
          platformColors: isVisible ? getPlatformColors(game) : null,
          key: game.gameDir || index,
        };
      }).filter(c => c.isVisible)
    : [];

  // Instant reactive active game name for title pill
  $: activeGame = count > 0 ? list[wrapIndex(Math.round(currentPos))] : null;

  let stickStopTimeout = null;

  // External step method for keyboard / D-pad
  export function step(delta) {
    if (count <= 1) return;
    targetPos = Math.round(targetPos) + delta;
  }

  // Rotary joystick scrubbing (iPod click wheel on Right Stick)
  export function addRotaryDelta(delta) {
    if (count <= 1) return;
    targetPos += delta;
    if (stickStopTimeout) clearTimeout(stickStopTimeout);
    stickStopTimeout = setTimeout(() => {
      targetPos = Math.round(targetPos);
    }, 100);
  }

  function handleCardClick(diff, game) {
    if (!game) return;
    if (Math.abs(diff) < 0.25) {
      dispatch('launch', game);
    } else {
      const stepDelta = Math.round(diff);
      targetPos = Math.round(currentPos) + stepDelta;
    }
  }

  // High-speed adaptive mouse wheel scrubbing with momentum acceleration
  function handleWheel(e) {
    e.preventDefault();
    if (count <= 1) return;

    const rawDelta = Math.abs(e.deltaY) >= Math.abs(e.deltaX) ? e.deltaY : e.deltaX;
    const now = performance.now();
    const dt = Math.max(1, now - lastWheelTime);

    // Discrete mouse wheel notch detection (deltaMode !== 0 or large delta jumps >= 40)
    const isDiscreteNotch = e.deltaMode !== 0 || Math.abs(rawDelta) >= 40;

    if (isDiscreteNotch) {
      const stepDir = rawDelta > 0 ? 1 : -1;
      // Adaptive acceleration: fast wheel spinning gains momentum (1x up to 5x speed)
      let speedBoost = 1;
      if (dt < 70) {
        speedBoost = Math.min(5, Math.floor(1 + (70 - dt) / 12));
      }
      targetPos = Math.round(targetPos) + (stepDir * speedBoost);
      lastWheelTime = now;
    } else {
      // Continuous touchpad / precision trackpad sweep
      wheelAccumulator += rawDelta;
      const threshold = 28;

      while (Math.abs(wheelAccumulator) >= threshold) {
        const stepDir = wheelAccumulator > 0 ? 1 : -1;
        targetPos = Math.round(targetPos) + stepDir;
        wheelAccumulator -= stepDir * threshold;
        lastWheelTime = now;
      }
    }

    if (wheelResetTimeout) clearTimeout(wheelResetTimeout);
    wheelResetTimeout = setTimeout(() => {
      wheelAccumulator = 0;
      targetPos = Math.round(targetPos);
    }, 100);
  }

  // Pointer / Touch drag scrubbing
  function handlePointerDown(e) {
    if (e.button !== 0 || count <= 1) return;
    isDragging = true;
    dragStartX = e.clientX;
    dragStartPos = currentPos;
    lastDragTime = performance.now();
    dragVelocity = 0;
  }

  function handlePointerMove(e) {
    if (!isDragging) return;
    const now = performance.now();
    const dt = Math.max(1, now - lastDragTime);
    const deltaPx = e.clientX - dragStartX;

    const newPos = dragStartPos - (deltaPx / 260);
    dragVelocity = (newPos - currentPos) / dt;
    currentPos = newPos;
    targetPos = newPos;
    lastDragTime = now;
  }

  function handlePointerUp() {
    if (!isDragging) return;
    isDragging = false;
    const projected = currentPos + dragVelocity * 80;
    targetPos = Math.round(projected);
  }

  // Smooth High-FPS Physics Loop
  function physicsLoop() {
    if (count > 0) {
      if (!isDragging) {
        const diff = targetPos - currentPos;
        if (Math.abs(diff) > 0.0005) {
          // Snappy 32% per frame tracking
          currentPos += diff * 0.32;
        } else {
          currentPos = targetPos;
        }
      }

      // Only notify heavy global store subscribers when settling close to destination,
      // preventing cascading DOM re-renders in drawers and HUD on every transit frame.
      const isSettling = Math.abs(targetPos - currentPos) < 0.15;
      if (isSettling) {
        const currentCenterIdx = wrapIndex(Math.round(currentPos));
        if (currentCenterIdx !== lastSyncIndex && list[currentCenterIdx]) {
          lastSyncIndex = currentCenterIdx;
          const targetGame = list[currentCenterIdx];
          lastEmittedDir = targetGame.gameDir;
          selectedGame.set(targetGame);
        }
      }
    }

    animFrameId = requestAnimationFrame(physicsLoop);
  }

  onMount(() => {
    if ($selectedGame && count > 0) {
      const dir = $selectedGame.gameDir;
      const idx = list.findIndex(g => isSamePath(g.gameDir, dir));
      if (idx >= 0) {
        currentPos = idx;
        targetPos = idx;
        lastSyncIndex = idx;
        lastEmittedDir = dir;
      }
    }
    animFrameId = requestAnimationFrame(physicsLoop);
  });

  onDestroy(() => {
    if (animFrameId) cancelAnimationFrame(animFrameId);
    if (wheelResetTimeout) clearTimeout(wheelResetTimeout);
    if (stickStopTimeout) clearTimeout(stickStopTimeout);
  });
</script>

<div
  bind:this={containerEl}
  role="region"
  aria-label="Game Carousel"
  on:wheel|nonpassive={handleWheel}
  on:pointerdown={handlePointerDown}
  on:pointermove={handlePointerMove}
  on:pointerup={handlePointerUp}
  on:pointercancel={handlePointerUp}
  class="relative w-full flex-1 flex flex-col items-center justify-center select-none overflow-visible touch-none my-auto"
  style="perspective: 1200px; perspective-origin: center 46%;"
>
  {#if count === 0}
    <div class="flex flex-col items-center justify-center p-8 text-center text-slate-400">
      <svg class="w-16 h-16 text-slate-600 mb-4 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
      <h3 class="text-lg font-bold text-slate-200 mb-1">No Games Found</h3>
      <p class="text-xs text-slate-400 max-w-sm">Configure your library paths or add titles to get started.</p>
    </div>
  {:else}
    <!-- 3D Curved Arc Carousel Stage -->
    <div class="relative w-full h-[400px] sm:h-[450px] md:h-[490px] flex items-center justify-center">
      {#each visibleCards as card (card.key)}
        {@const style = card.style}
        {#if style}
          <div
            on:click={() => handleCardClick(card.diff, card.game)}
            role="button"
            tabindex="0"
            on:keydown={(e) => e.key === 'Enter' && handleCardClick(card.diff, card.game)}
            class="absolute top-1/2 left-1/2 cursor-pointer will-change-transform"
            style="
              transform: {style.transform};
              z-index: {style.zIndex};
            "
          >
            <!-- Card Container (100% GPU-composited transform and opacity) -->
            <div
              class="relative w-52 h-[340px] sm:w-60 sm:h-[395px] md:w-68 md:h-[445px] rounded-[32px] overflow-hidden shadow-2xl bg-[#161622]"
              style="opacity: {style.opacity};"
            >
              <!-- Poster Artwork (Stable per game, never unmounts while in view) -->
              <PosterArt
                appId={card.game.appId}
                name={card.game.name}
                posterUrl={card.game.coverUrl}
                localCoverUrl={card.game.localCoverUrl}
                className="w-full h-full object-cover select-none pointer-events-none"
              />

              <!-- Hardware-accelerated GPU Black Dimmer for depth falloff (0% CPU cost) -->
              <div
                class="absolute inset-0 bg-black pointer-events-none"
                style="opacity: {style.dimOpacity};"
              ></div>

              <!-- Active Glow Border Overlay (Fades in purely via hardware opacity) -->
              <div
                class="absolute inset-0 rounded-[32px] pointer-events-none border-2 transition-opacity"
                style="
                  border-color: {card.platformColors?.border || '#1a9fff'};
                  box-shadow: 0 0 25px -3px {card.platformColors?.glow || 'rgba(26, 159, 255, 0.4)'};
                  opacity: {style.centerProximity > 0.3 ? style.centerProximity : 0};
                "
              ></div>

              <!-- Platform Pill -->
              {#if card.platformColors}
                <div
                  class="absolute top-3.5 left-3.5 px-3 py-1 rounded-full text-[10px] sm:text-[11px] font-black uppercase tracking-wider text-white shadow-xl flex items-center space-x-1.5 pointer-events-none z-10 bg-slate-900/90"
                  style="
                    border: 1px solid {card.platformColors.border};
                    opacity: {style.centerProximity};
                  "
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-white animate-pulse"></span>
                  <span>{card.platformColors.label}</span>
                </div>
              {/if}

              <!-- Running Status Pill -->
              {#if card.game.isRunning}
                <div
                  class="absolute top-3.5 right-3.5 px-2.5 py-1 rounded-full bg-emerald-500 text-slate-950 text-[10px] font-black uppercase tracking-wider shadow-lg flex items-center space-x-1 animate-pulse pointer-events-none z-10"
                  style="opacity: {style.centerProximity};"
                >
                  <span>Running</span>
                </div>
              {/if}
            </div>
          </div>
        {/if}
      {/each}
    </div>

    <!-- Active Game Floating Title Badge Below Card -->
    {#if activeGame}
      <div class="mt-3 sm:mt-5 flex items-center justify-center z-40 transition-all duration-200 ease-out">
        <div class="px-6 py-1.5 rounded-full bg-white/95 text-slate-950 font-black text-sm sm:text-base tracking-tight shadow-2xl flex items-center space-x-2 max-w-md truncate border border-white/50 backdrop-blur-md">
          <span class="truncate">{activeGame.name}</span>
        </div>
      </div>
    {/if}
  {/if}
</div>
