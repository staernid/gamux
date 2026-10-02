<script>
  /**
   * @typedef {Object} Props
   * @property {number} [count]
   * @property {number} [currentPos]
   */

  /** @type {Props} */
  let { count = 0, currentPos = 0 } = $props();

  // Higher/deeper and slightly wider upward convex arch curve
  // Span: 250px width (20 to 270), Height rise: 46px (from y=58 up to y=12)
  const p0 = { x: 20, y: 58 };
  const p1 = { x: 145, y: -34 };
  const p2 = { x: 270, y: 58 };

  const pathD = `M ${p0.x},${p0.y} Q ${p1.x},${p1.y} ${p2.x},${p2.y}`;

  // Normalized progress [0, 1]
  let progress = $derived((() => {
    if (count <= 1) return 0.5;
    const wrapped = ((currentPos % count) + count) % count;
    return wrapped / (count - 1);
  })());

  // Thick, chunky, extremely rounded curved pill (pathLength=100)
  const thumbLength = 18; // Length of the curved capsule
  let dashOffset = $derived(-(progress * (100 - thumbLength)));
</script>

{#if count > 1}
  <div
    role="presentation"
    aria-hidden="true"
    class="relative px-2 py-0.5 flex items-center justify-center select-none pointer-events-none"
  >
    <svg
      viewBox="0 0 290 64"
      class="w-64 sm:w-72 md:w-80 h-12 overflow-visible drop-shadow-[0_2px_4px_rgba(0,0,0,0.85)]"
    >
      <!-- Thick, Extremely Rounded Solid White Curved Pill -->
      <path
        d={pathD}
        fill="none"
        stroke="#ffffff"
        stroke-width="11"
        stroke-linecap="round"
        pathLength="100"
        stroke-dasharray="{thumbLength} 100"
        stroke-dashoffset={dashOffset}
      />

      <!-- Crisp Specular Highlight Core -->
      <path
        d={pathD}
        fill="none"
        stroke="#f8fafc"
        stroke-width="7"
        stroke-linecap="round"
        pathLength="100"
        stroke-dasharray="{thumbLength - 4} 100"
        stroke-dashoffset={dashOffset - 2}
      />
    </svg>
  </div>
{/if}
