<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import PosterArt from '../common/PosterArt.svelte';
  import ArchScrollbar from './ArchScrollbar.svelte';
  import { filteredGames, selectedGame, runningGames, isGamePathRunning } from '../../stores/library.js';
  import { getPlatformColors, isSamePath, extractCoverColorPalette, getCachedCoverPalette, formatPlaytime } from '../../utils/game.js';
  import { playTick, playSelect } from '../../utils/audio.js';

  const dispatch = createEventDispatcher();

  let containerEl;
  let animFrameId = null;
  let isLoopRunning = false;

  function wakeLoop() {
    if (!isLoopRunning && typeof window !== 'undefined') {
      isLoopRunning = true;
      lastPhysicsTime = performance.now();
      animFrameId = requestAnimationFrame(physicsLoop);
    }
  }

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
  let lastSoundIndex = -1;
  let lastList = null;

  $: list = $filteredGames || [];
  $: count = list.length;

  // Synchronize when selectedGame changes externally (e.g. keyboard navigation, search grid, drawer, or sort)
  $: if ($selectedGame && count > 0) {
    const dir = $selectedGame.gameDir;
    const listChanged = list !== lastList;
    if (dir !== lastEmittedDir || listChanged) {
      lastEmittedDir = dir;
      lastList = list;
      const idx = list.findIndex(g => isSamePath(g.gameDir, dir));
      if (idx >= 0 && !isDragging) {
        const currWrapped = wrapIndex(Math.round(currentPos));
        let diff = idx - currWrapped;
        if (diff > count / 2) diff -= count;
        if (diff < -count / 2) diff += count;
        targetPos = Math.round(currentPos) + diff;
        wakeLoop();
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

  // Pointer / Mouse Interactive Tilt State
  let pointerNormX = 0;
  let pointerNormY = 0;
  let currentTiltX = 0;
  let currentTiltY = 0;
  let isHovering = false;

  // Smooth continuous 3D card layout along a spatial depth arc (Pure GPU properties only)
  function getCardTransform(diff) {
    const absDiff = Math.abs(diff);

    // Exact center settled card with interactive 3D mouse parallax tilt
    if (absDiff < 0.0001) {
      return {
        transform: `translate(-50%, -50%) translate3d(0px, 0px, 0px) rotateX(${currentTiltY.toFixed(2)}deg) rotateY(${currentTiltX.toFixed(2)}deg) scale(1)`,
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
    const baseRotY = -sign * Math.min(42, Math.pow(absDiff, 0.85) * 24);

    // Near-center interactive tilt contribution
    const centerFactor = Math.max(0, 1 - absDiff * 1.8);
    const rotY = baseRotY + (currentTiltX * centerFactor);
    const rotX = currentTiltY * centerFactor;

    // Scale falloff
    const scale = Math.max(0.65, 1 - absDiff * 0.12);

    // Opacity and black dimmer for depth falloff
    const opacity = Math.max(0, 1 - Math.pow(absDiff / 2.6, 1.6));
    const dimOpacity = Math.min(0.65, absDiff * 0.3);

    return {
      transform: `translate(-50%, -50%) translate3d(${x.toFixed(1)}px, 0px, ${z.toFixed(1)}px) rotateX(${rotX.toFixed(2)}deg) rotateY(${rotY.toFixed(2)}deg) scale(${scale.toFixed(3)})`,
      zIndex: Math.round(100 - absDiff * 20),
      opacity: Math.max(0, Math.min(1, opacity)),
      dimOpacity: dimOpacity,
      centerProximity: Math.max(0, 1 - absDiff * 1.5),
    };
  }

  // Render only 5 visible active games within view window with stable DOM identity (O(1) instead of O(N) library scan)
  $: visibleCards = (() => {
    if (count === 0) return [];
    if (count <= 5) {
      return list.map((game, index) => {
        const diff = getGameDiff(index, currentPos, count);
        return {
          game,
          index,
          diff,
          style: getCardTransform(diff),
          platformColors: getPlatformColors(game),
          key: game.gameDir || index,
        };
      });
    }

    const baseIdx = Math.round(currentPos);
    const cards = [];
    for (let offset = -2; offset <= 2; offset++) {
      const idx = wrapIndex(baseIdx + offset);
      const game = list[idx];
      if (game) {
        const diff = getGameDiff(idx, currentPos, count);
        cards.push({
          game,
          index: idx,
          diff,
          style: getCardTransform(diff),
          platformColors: getPlatformColors(game),
          key: game.gameDir || idx,
        });
      }
    }
    return cards;
  })();

  // Instant reactive active game name for title pill
  $: activeGame = count > 0 ? list[wrapIndex(Math.round(currentPos))] : null;
  $: activeColors = activeGame ? getPlatformColors(activeGame) : null;

  let activePalette = [];
  $: activeCoverUrl = activeGame?.localCoverUrl || activeGame?.coverUrl || '';

  $: if (activeCoverUrl) {
    const cached = getCachedCoverPalette(activeCoverUrl);
    if (cached && cached.length > 0) {
      activePalette = cached;
    } else {
      extractCoverColorPalette(activeCoverUrl).then(pal => {
        if (pal && pal.length > 0 && (activeGame?.localCoverUrl === activeCoverUrl || activeGame?.coverUrl === activeCoverUrl)) {
          activePalette = pal;
        }
      });
    }
  } else {
    activePalette = [];
  }

  function parseRgb(colorStr) {
    if (!colorStr || typeof colorStr !== 'string') return { r: 26, g: 159, b: 255 };
    const m = colorStr.match(/\d+/g);
    if (m && m.length >= 3) {
      return { r: Number(m[0]), g: Number(m[1]), b: Number(m[2]) };
    }
    return { r: 160, g: 160, b: 165 };
  }

  // Reactive Ambient Glow derived directly from active game palette
  $: glowRgb = (() => {
    const pal = activePalette.length > 0 ? activePalette : (activeColors?.border ? [activeColors.border] : []);
    if (pal.length > 0) {
      const c = parseRgb(pal[0]);
      return `${c.r}, ${c.g}, ${c.b}`;
    }
    return '160, 160, 165';
  })();

  let stickStopTimeout = null;

  // External step method for keyboard / D-pad
  export function step(delta) {
    if (count <= 1) return;
    targetPos = Math.round(targetPos) + delta;
    lastSoundIndex = wrapIndex(Math.round(targetPos));
    playTick();
    wakeLoop();
  }

  // Rotary joystick scrubbing (iPod click wheel on Right Stick)
  export function addRotaryDelta(delta) {
    if (count <= 1) return;
    targetPos += delta;
    wakeLoop();
    if (stickStopTimeout) clearTimeout(stickStopTimeout);
    stickStopTimeout = setTimeout(() => {
      targetPos = Math.round(targetPos);
      wakeLoop();
    }, 100);
  }

  // External seek method for arch scrollbar / direct positioning
  export function seek(targetIdx) {
    handleSeek({ detail: targetIdx });
  }

  export function getCurrentPos() {
    return currentPos;
  }

  function handleSeek(e) {
    const targetIdx = e.detail;
    if (count <= 1 || targetIdx < 0 || targetIdx >= count) return;
    const currWrapped = wrapIndex(Math.round(currentPos));
    let diff = targetIdx - currWrapped;
    if (diff > count / 2) diff -= count;
    if (diff < -count / 2) diff += count;
    targetPos = Math.round(currentPos) + diff;
    lastSoundIndex = targetIdx;
    playTick();
    wakeLoop();
  }

  function handleCardClick(diff, game) {
    if (!game) return;
    if (Math.abs(diff) < 0.25) {
      playSelect();
      dispatch('launch', game);
    } else {
      const stepDelta = Math.round(diff);
      targetPos = Math.round(currentPos) + stepDelta;
      wakeLoop();
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
      wakeLoop();
    } else {
      // Continuous touchpad / precision trackpad sweep
      wheelAccumulator += rawDelta;
      const threshold = 28;

      while (Math.abs(wheelAccumulator) >= threshold) {
        const stepDir = wheelAccumulator > 0 ? 1 : -1;
        targetPos = Math.round(targetPos) + stepDir;
        wheelAccumulator -= stepDir * threshold;
        lastWheelTime = now;
        wakeLoop();
      }
    }

    if (wheelResetTimeout) clearTimeout(wheelResetTimeout);
    wheelResetTimeout = setTimeout(() => {
      wheelAccumulator = 0;
      targetPos = Math.round(targetPos);
      wakeLoop();
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
    wakeLoop();
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
    wakeLoop();
  }

  function handlePointerUp() {
    if (!isDragging) return;
    isDragging = false;
    const projected = currentPos + dragVelocity * 80;
    targetPos = Math.round(projected);
    wakeLoop();
  }

  function handleContainerPointerMove(e) {
    if (isDragging) {
      handlePointerMove(e);
      return;
    }
    if (!containerEl) return;
    const rect = containerEl.getBoundingClientRect();
    if (rect.width > 0 && rect.height > 0) {
      pointerNormX = ((e.clientX - rect.left) / rect.width - 0.5) * 2;
      pointerNormY = ((e.clientY - rect.top) / rect.height - 0.5) * 2;
      isHovering = true;
      wakeLoop();
    }
  }

  function handleContainerPointerLeave() {
    isHovering = false;
    pointerNormX = 0;
    pointerNormY = 0;
    wakeLoop();
  }

  let lastPhysicsTime = 0;
  let springVelocity = 0;

  // Smooth Frame-Rate Independent Spring Physics Loop
  function physicsLoop(timestamp) {
    if (count > 0) {
      if (!lastPhysicsTime) lastPhysicsTime = timestamp || performance.now();
      const now = timestamp || performance.now();
      const dt = Math.min(64, Math.max(1, now - lastPhysicsTime)) / 1000;
      lastPhysicsTime = now;

      let positionSettled = false;
      if (!isDragging) {
        const diff = targetPos - currentPos;
        if (Math.abs(diff) > 0.0001 || Math.abs(springVelocity) > 0.0001) {
          // Tactile spring physics with subtle settle snap
          const stiffness = 210;
          const damping = 23;
          const springForce = diff * stiffness - springVelocity * damping;
          springVelocity += springForce * dt;
          currentPos += springVelocity * dt;
        } else {
          currentPos = targetPos;
          springVelocity = 0;
          positionSettled = true;
        }
      } else {
        springVelocity = 0;
      }

      // Smoothly interpolate interactive card tilt (lag-free critically damped follower)
      const targetTiltX = isHovering && !isDragging ? pointerNormX * 7.5 : 0;
      const targetTiltY = isHovering && !isDragging ? -pointerNormY * 6.0 : 0;
      const tiltFactor = 1 - Math.exp(-18 * dt);
      const diffTiltX = targetTiltX - currentTiltX;
      const diffTiltY = targetTiltY - currentTiltY;

      let tiltSettled = false;
      if (Math.abs(diffTiltX) > 0.005 || Math.abs(diffTiltY) > 0.005) {
        currentTiltX += diffTiltX * tiltFactor;
        currentTiltY += diffTiltY * tiltFactor;
      } else {
        currentTiltX = targetTiltX;
        currentTiltY = targetTiltY;
        tiltSettled = true;
      }

      // Central mechanical tick on crossing card index boundaries
      const currentCenterIdx = wrapIndex(Math.round(currentPos));
      if (lastSoundIndex !== -1 && currentCenterIdx !== lastSoundIndex) {
        lastSoundIndex = currentCenterIdx;
        playTick();
      } else if (lastSoundIndex === -1 && count > 0) {
        lastSoundIndex = currentCenterIdx;
      }

      // Only notify heavy global store subscribers when settling close to destination,
      // preventing cascading DOM re-renders in drawers and HUD on every transit frame.
      const isSettling = Math.abs(targetPos - currentPos) < 0.15;
      if (isSettling) {
        if (currentCenterIdx !== lastSyncIndex && list[currentCenterIdx]) {
          lastSyncIndex = currentCenterIdx;
          const targetGame = list[currentCenterIdx];
          lastEmittedDir = targetGame.gameDir;
          selectedGame.set(targetGame);
        }
      }

      // Sleep loop if motion and tilt are completely settled and not dragging
      if (positionSettled && tiltSettled && !isDragging) {
        isLoopRunning = false;
        animFrameId = null;
        lastPhysicsTime = 0;
        return;
      }
    } else {
      isLoopRunning = false;
      animFrameId = null;
      lastPhysicsTime = 0;
      return;
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
        lastSoundIndex = idx;
        lastEmittedDir = dir;
      } else {
        currentPos = 0;
        targetPos = 0;
        lastSyncIndex = 0;
        lastSoundIndex = 0;
        if (list[0]) {
          lastEmittedDir = list[0].gameDir;
          selectedGame.set(list[0]);
        }
      }
    } else if (count > 0) {
      currentPos = 0;
      targetPos = 0;
      lastSyncIndex = 0;
      lastSoundIndex = 0;
      if (list[0]) {
        lastEmittedDir = list[0].gameDir;
        selectedGame.set(list[0]);
      }
    }
    wakeLoop();
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
  on:pointermove={handleContainerPointerMove}
  on:pointerleave={handleContainerPointerLeave}
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
    <div class="relative w-full h-[400px] sm:h-[450px] md:h-[490px] flex items-center justify-center contain-paint">
      <!-- Ambient Dynamic Cover Radial Glow (GPU Composited with smooth CSS transition) -->
      <div
        class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[750px] sm:w-[900px] md:w-[1050px] lg:w-[1150px] h-[480px] sm:h-[560px] md:h-[620px] rounded-full pointer-events-none z-0 transition-[background] duration-700 ease-out"
        style="background: radial-gradient(ellipse at center, rgba({glowRgb}, 0.48) 0%, rgba({glowRgb}, 0.18) 36%, rgba({glowRgb}, 0.03) 62%, transparent 75%);"
      ></div>

      <!-- Physical Stage Floor Contact Shadow (Grounding the 3D cards) -->
      <div
        class="absolute bottom-1 sm:bottom-[-2px] left-1/2 -translate-x-1/2 w-[240px] sm:w-[280px] md:w-[320px] h-[26px] rounded-full pointer-events-none z-0"
        style="background: radial-gradient(ellipse at center, rgba(0,0,0,0.85) 0%, rgba(0,0,0,0.45) 45%, transparent 75%);"
      ></div>

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
              transform-origin: 50% 100%;
            "
          >
            <!-- Card Container (100% GPU-composited transform and opacity, strict 2:3 poster aspect ratio) -->
            <div
              class="relative h-[340px] sm:h-[395px] md:h-[450px] aspect-[2/3] rounded-[32px] overflow-hidden shadow-xl bg-[#161622] border border-white/10"
              style="opacity: {style.opacity};"
            >
              <!-- Poster Artwork (Stable per game, never unmounts while in view) -->
              <PosterArt
                appId={card.game.appId}
                name={card.game.name}
                posterUrl={card.game.coverUrl}
                localCoverUrl={card.game.localCoverUrl}
                thumbnailWidth={350}
                className="w-full h-full object-cover select-none pointer-events-none"
              />

              <!-- Top-edge Glass Rim Specular Highlight (Catching stage lighting) -->
              <div class="absolute inset-x-0 top-0 h-[2px] bg-gradient-to-r from-transparent via-white/50 to-transparent pointer-events-none z-20"></div>

              <!-- Hardware-accelerated GPU Black Dimmer for depth falloff (0% CPU cost) -->
              <div
                class="absolute inset-0 bg-black pointer-events-none"
                style="opacity: {style.dimOpacity};"
              ></div>
            </div>
          </div>
        {/if}
      {/each}
    </div>

    <!-- Active Game Floating Title Badge Below Card -->
    {#if activeGame}
      <div class="mt-2 sm:mt-4 flex items-center justify-center z-40 transition-opacity duration-200 ease-out">
        <div class="px-5 py-1.5 rounded-full bg-neutral-900/90 text-white font-bold text-sm sm:text-base tracking-tight shadow-2xl flex items-center justify-center gap-2 max-w-md truncate border border-white/20 hover:border-white/40 transition-all">
          {#if isGamePathRunning($runningGames, activeGame.gameDir)}
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse flex-shrink-0"></span>
          {/if}
          <span class="truncate">{activeGame.name}</span>
          {#if activeGame.playtimeSeconds > 0}
            <span class="text-2xs font-mono text-neutral-400 font-normal flex-shrink-0">· {formatPlaytime(activeGame.playtimeSeconds)}</span>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Synchronized 60/120fps Momentum Arch Positional Widget -->
    <div class="absolute bottom-1.5 left-1/2 -translate-x-1/2 z-30 pointer-events-none">
      <ArchScrollbar {count} {currentPos} />
    </div>
  {/if}
</div>
