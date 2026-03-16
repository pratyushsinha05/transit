/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    theme: {
        extend: {
            colors: {
                hud: {
                    bg: '#0a0e1a',
                    panel: '#0d1321',
                    surface: '#111827',
                    border: '#1a2332',
                    'border-glow': 'rgba(0, 245, 212, 0.15)',
                    accent: '#00f5d4',
                    'accent-dim': 'rgba(0, 245, 212, 0.4)',
                    warn: '#ff6b35',
                    danger: '#ff3860',
                    text: '#c8d6e5',
                    'text-dim': '#576574',
                    'text-bright': '#f0f4f8',
                },
            },
            fontFamily: {
                mono: ['"Space Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
            },
            letterSpacing: {
                'hud': '0.12em',
                'hud-wide': '0.2em',
            },
            keyframes: {
                'radar-ping': {
                    '0%': { transform: 'scale(0.5)', opacity: '1' },
                    '100%': { transform: 'scale(3)', opacity: '0' },
                },
                'blink': {
                    '0%, 100%': { opacity: '1' },
                    '50%': { opacity: '0.2' },
                },
                'glow-pulse': {
                    '0%, 100%': { boxShadow: '0 0 4px rgba(0, 245, 212, 0.3)' },
                    '50%': { boxShadow: '0 0 12px rgba(0, 245, 212, 0.6)' },
                },
                'scanline': {
                    '0%': { transform: 'translateY(-100%)' },
                    '100%': { transform: 'translateY(100%)' },
                },
            },
            animation: {
                'radar-ping': 'radar-ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite',
                'blink': 'blink 1.2s ease-in-out infinite',
                'glow-pulse': 'glow-pulse 2s ease-in-out infinite',
                'scanline': 'scanline 8s linear infinite',
            },
        },
    },
    plugins: [],
}
