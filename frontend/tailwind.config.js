/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}'
  ],
  theme: {
    extend: {
      borderRadius: {
        xl2: '1.25rem',
        xl3: '1.75rem'
      },
      boxShadow: {
        glass: '0 20px 60px rgba(2, 6, 23, 0.35)',
        soft: '0 10px 30px rgba(2, 6, 23, 0.18)'
      },
      colors: {
        ui: {
          canvas: 'var(--bg-canvas)',
          surface: 'var(--bg-surface)',
          text: 'var(--text-primary)',
          muted: 'var(--text-muted)',
          accent: 'var(--accent-strong)',
          success: 'var(--success)',
          warning: 'var(--warning)',
          danger: 'var(--danger)'
        }
      },
      transitionTimingFunction: {
        smooth: 'cubic-bezier(0.4, 0, 0.2, 1)'
      }
    }
  },
  plugins: []
};
