/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx,html,svelte}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      fontSize: {
        '2xs': ['0.6875rem', { lineHeight: '0.95rem' }], // ~11px at base, scales dynamically with root font size
        '3xs': ['0.625rem', { lineHeight: '0.85rem' }],  // ~10px at base
        '4xs': ['0.5625rem', { lineHeight: '0.75rem' }],  // ~9px at base
      },
      colors: {
        surface: {
          base: 'rgb(var(--surface-base) / <alpha-value>)',
          header: 'rgb(var(--surface-header) / <alpha-value>)',
          card: 'rgb(var(--surface-card) / <alpha-value>)',
          cardHover: 'rgb(var(--surface-card-hover) / <alpha-value>)',
          elevated: 'rgb(var(--surface-elevated) / <alpha-value>)',
          border: 'rgb(var(--surface-border) / <alpha-value>)',
          borderSubtle: 'rgb(var(--surface-border-subtle) / <alpha-value>)',
          borderHighlight: 'rgb(var(--surface-border-highlight) / <alpha-value>)',
          input: 'rgb(var(--surface-input) / <alpha-value>)',
          pill: 'rgb(var(--surface-pill) / <alpha-value>)',
        }
      }
    },
  },
  plugins: [],
}
