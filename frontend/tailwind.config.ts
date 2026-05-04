import type { Config } from 'tailwindcss'

export default {
  content: ['./app/**/*.{vue,ts,js}'],
  theme: {
    extend: {
      colors: {
        accent: {
          DEFAULT: '#1877F2',
          hover: '#1464D8',
          soft: '#E7F0FE',
          deep: '#4267B2',
        },
        bg: {
          DEFAULT: '#FFFFFF',
          subtle: '#F0F2F5',
          muted: '#F7F8FA',
        },
        ink: {
          DEFAULT: '#050505',
          muted: '#65676B',
          faint: '#8A8D91',
        },
        border: {
          DEFAULT: '#DADDE1',
          strong: '#CED0D4',
        },
        success: { DEFAULT: '#42B72A', soft: '#E6F4EA' },
        warning: { DEFAULT: '#F7B928', soft: '#FEF6E0' },
        danger: { DEFAULT: '#E41E3F', soft: '#FCEBED' },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'Segoe UI', 'Roboto', 'sans-serif'],
      },
      borderRadius: {
        md: '0.5rem',
        lg: '0.625rem',
        xl: '0.875rem',
        '2xl': '1.25rem',
      },
      boxShadow: {
        sm: '0 1px 2px 0 rgb(0 0 0 / 0.04)',
        DEFAULT: '0 1px 3px 0 rgb(0 0 0 / 0.06), 0 1px 2px -1px rgb(0 0 0 / 0.04)',
        focus: '0 0 0 3px rgb(24 119 242 / 0.25)',
      },
    },
  },
  plugins: [],
} satisfies Config
