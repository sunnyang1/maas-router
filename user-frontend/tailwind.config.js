/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    './pages/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        // WorldACLaw.ai inspired palette
        bg: {
          primary: '#0c0c0c',
          secondary: '#141414',
          tertiary: '#1a1a1a',
          elevated: '#1f1f1f',
        },
        text: {
          primary: '#f5f5f5',
          secondary: '#a1a1a1',
          tertiary: '#737373',
          muted: '#525252',
        },
        accent: {
          DEFAULT: '#ff6b35',
          hover: '#ff8555',
          muted: 'rgba(255, 107, 53, 0.2)',
        },
        'accent-2': {
          DEFAULT: '#14b8a6',
          hover: '#2dd4bf',
          muted: 'rgba(20, 184, 166, 0.2)',
        },
      },
      fontFamily: {
        sans: ['Plus Jakarta Sans', '-apple-system', 'BlinkMacSystemFont', 'sans-serif'],
        display: ['Space Grotesk', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      fontSize: {
        'display-1': ['4.5rem', { lineHeight: '1.1', letterSpacing: '-0.02em' }],
        'display-2': ['3.5rem', { lineHeight: '1.15', letterSpacing: '-0.02em' }],
        'display-3': ['2.5rem', { lineHeight: '1.2', letterSpacing: '-0.02em' }],
        'heading-1': ['2rem', { lineHeight: '1.25', letterSpacing: '-0.01em' }],
        'heading-2': ['1.5rem', { lineHeight: '1.3', letterSpacing: '-0.01em' }],
        'heading-3': ['1.25rem', { lineHeight: '1.4' }],
        'body-large': ['1.125rem', { lineHeight: '1.6' }],
        'body': ['1rem', { lineHeight: '1.6' }],
        'body-small': ['0.875rem', { lineHeight: '1.5' }],
        'caption': ['0.75rem', { lineHeight: '1.5' }],
      },
      animation: {
        'fade-in-up': 'fadeInUp 0.6s ease-out forwards',
        'fade-in': 'fadeIn 0.5s ease-out forwards',
        'slide-in': 'slideIn 0.5s ease-out forwards',
        'pulse-glow': 'pulseGlow 2s ease-in-out infinite',
        'float': 'float 3s ease-in-out infinite',
      },
      keyframes: {
        fadeInUp: {
          '0%': { opacity: '0', transform: 'translateY(30px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideIn: {
          '0%': { opacity: '0', transform: 'translateX(-20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        pulseGlow: {
          '0%, 100%': { boxShadow: '0 0 20px rgba(255, 107, 53, 0.3)' },
          '50%': { boxShadow: '0 0 40px rgba(255, 107, 53, 0.5)' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-10px)' },
        },
      },
      backgroundImage: {
        'gradient-hero': 'linear-gradient(135deg, rgba(255, 107, 53, 0.15) 0%, rgba(20, 184, 166, 0.1) 50%, transparent 100%)',
        'gradient-card': 'linear-gradient(145deg, rgba(255, 255, 255, 0.05) 0%, transparent 100%)',
        'gradient-border': 'linear-gradient(135deg, rgba(255, 107, 53, 0.5) 0%, rgba(20, 184, 166, 0.3) 100%)',
        'gradient-radial': 'radial-gradient(ellipse at top, rgba(255, 107, 53, 0.1) 0%, transparent 50%), radial-gradient(ellipse at bottom, rgba(20, 184, 166, 0.05) 0%, transparent 50%)',
      },
      boxShadow: {
        'glow': '0 0 30px rgba(255, 107, 53, 0.3)',
        'glow-lg': '0 0 50px rgba(255, 107, 53, 0.4)',
        'card': '0 4px 20px rgba(0, 0, 0, 0.3)',
        'card-hover': '0 20px 40px -15px rgba(255, 107, 53, 0.2)',
      },
      borderRadius: {
        'xl': '12px',
        '2xl': '16px',
        '3xl': '24px',
      },
    },
  },
  plugins: [],
}
