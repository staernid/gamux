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
          base: '#09090b',
          header: '#121214',
          card: '#18181b',
          cardHover: '#222225',
          elevated: '#202024',
          border: '#27272a',
          borderSubtle: '#1f1f23',
          borderHighlight: '#3f3f46',
          pill: '#222226',
        }
      }
    },
  },
  plugins: [],
}
