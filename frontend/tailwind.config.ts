import type { Config } from 'tailwindcss'

export default {
  content: [
    './app/**/*.{vue,ts,js}',
    './components/**/*.{vue,ts,js}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // ─── Design System Palette ───────────────────────────────────────────
        // Background scale (dark navy → near-black)
        'bg-base':     '#04080F',
        'bg-surface':  '#0A1628',
        'bg-elevated': '#0F2040',
        'bg-border':   '#1A3558',

        // Blue accent scale
        'blue-muted':   '#1E4D8C',
        'blue-default': '#2563EB',
        'blue-bright':  '#3B82F6',
        'blue-glow':    '#60A5FA',

        // Status colors
        'status-scaling': '#10B981',
        'status-healthy': '#3B82F6',
        'status-atrisk':  '#F59E0B',
        'status-under':   '#EF4444',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      borderRadius: {
        'card':   '16px',
        'button': '12px',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
    },
  },
  plugins: [],
} satisfies Config
