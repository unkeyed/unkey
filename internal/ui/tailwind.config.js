/** @type {import('tailwindcss').Config} */
export default {
  content: [],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-geist-sans)"],
        mono: ["var(--font-geist-mono)"],
      },
      colors: {
        black: "black",
        white: "white",
        transparent: "transparent",
        current: "currentColor",
        gray: palette("gray"),
        info: palette("info"),
        success: palette("success"),
        warning: palette("warning"),
        error: palette("error"),
        feature: palette("feature"),
        accent: palette("accent"),
      },
      dropShadow: {
        // from vitor's figma
        button: "0px 4px 4px 0px rgba(0, 0, 0, 0.25)",
      },
    },
  },
  plugins: [],
};

/**
 * returns a 12 step color scale from css variables
 *
 * @example:
 * {
 *   1: "hsl(var(--gray-1))",
 *   2: "hsl(var(--gray-2))",
 *   ...,
 *   12: "hsl(var(--gray-12))",
 * }
 */
function palette(name) {
  const colors = {};
  for (let i = 1; i <= 12; i++) {
    colors[i] = `hsl(var(--${name}-${i}))`;
  }
  return colors;
}
