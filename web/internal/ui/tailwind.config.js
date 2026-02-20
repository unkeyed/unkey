/**
 * Tailwind v4 Configuration for @unkey/ui
 *
 * In Tailwind v4, configuration is primarily done in CSS using @theme blocks.
 * This JavaScript config is kept for backwards compatibility and tooling support.
 *
 * Apps consuming @unkey/ui should:
 * 1. Import "@unkey/ui/css" for color definitions
 * 2. Configure their own Tailwind v4 CSS-first setup
 * 3. Reference UI colors in their @theme blocks
 */

/** @type {import('tailwindcss').Config} */
export default {
  content: [],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-geist-sans)"],
        mono: ["var(--font-geist-mono)"],
      },
    },
  },
  plugins: [],
};
