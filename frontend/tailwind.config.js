/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx,html,svelte}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        surface: {
          base: '#101013',
          header: '#16161a',
          card: '#1a1a20',
          cardHover: '#22222a',
          elevated: '#1e1e26',
          border: '#2c2c36',
          borderSubtle: '#22222a',
          pill: '#24242e',
        }
      }
    },
  },
  plugins: [],
}
