/** @type {import('tailwindcss').Config} */
const defaultTheme = require("tailwindcss/defaultTheme");
const _plugin = require("tailwindcss/plugin");

module.exports = {
  darkMode: ["class"],
  content: [
    "./pages/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
    "./src/**/*.{ts,tsx}",
  ],
  theme: {
    fontSize: {
      xs: ["0.75rem", { lineHeight: "1rem" }],
      sm: ["0.875rem", { lineHeight: "1.5rem" }],
      base: ["1rem", { lineHeight: "1.75rem" }],
      lg: ["1.125rem", { lineHeight: "1.75rem" }],
      xl: ["1.25rem", { lineHeight: "2rem" }],
      "2xl": ["1.5rem", { lineHeight: "2.25rem" }],
      "3xl": ["1.75rem", { lineHeight: "2.25rem" }],
      "4xl": ["2rem", { lineHeight: "2.5rem" }],
      "5xl": ["2.5rem", { lineHeight: "3rem" }],
      "6xl": ["3rem", { lineHeight: "3.5rem" }],
      "7xl": ["4rem", { lineHeight: "4.5rem" }],
    },
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },

    colors: {
      current: "currentColor",
      transparent: "transparent",
      white: "hsl(var(--white))",
      black: "hsl(var(--black))",
      gray: {
        50: "#fafaf9",
        100: "#f5f5f4",
        200: "#e7e5e4",
        300: "#d6d3d1",
        400: "#a8a29e",
        500: "#78716c",
        600: "#57534e",
        700: "#44403c",
        800: "#292524",
        900: "#1c1917",
        950: "#0c0a09",
      },
      background: {
        DEFAULT: "hsl(var(--background))",
        subtle: "hsl(var(--background-subtle))",
      },
      content: {
        DEFAULT: "hsl(var(--content))",
        subtle: "hsl(var(--content-subtle))",
        info: "hsl(var(--content-info))",
        warn: "hsl(var(--content-warn))",
        alert: "hsl(var(--content-alert))",
      },

      brand: {
        DEFAULT: "hsl(var(--brand))",
        foreground: "hsl(var(--brand-foreground))",
      },

      warn: {
        DEFAULT: "hsl(var(--warn))",
        foreground: "hsl(var(--warn-foreground))",
      },

      alert: {
        DEFAULT: "hsl(var(--alert))",
        foreground: "hsl(var(--alert-foreground))",
      },

      subtle: {
        DEFAULT: "hsl(var(--subtle))",
        foreground: "hsl(var(--subtle-foreground))",
      },

      primary: {
        DEFAULT: "hsl(var(--primary))",
        foreground: "hsl(var(--primary-foreground))",
      },

      secondary: {
        DEFAULT: "hsl(var(--secondary))",
        foreground: "hsl(var(--secondary-foreground))",
      },

      border: "hsl(var(--border))",
      ring: "hsl(var(--ring))",
    },
    extend: {
      keyframes: {
        "accordion-down": {
          from: { height: 0 },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: 0 },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
      },
      fontFamily: {
        sans: ["Cal Sans", ...defaultTheme.fontFamily.sans],
        display: [
          ["Cal Sans", ...defaultTheme.fontFamily.sans],
          { fontVariationSettings: '"wdth" 125' },
        ],
      },
    },
  },
  plugins: [
    require("tailwindcss-animate"),
    require("@tailwindcss/typography"),
    require("@tailwindcss/aspect-ratio"),
  ],
};
