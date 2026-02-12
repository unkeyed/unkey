/** @type {import('tailwindcss').Config} */

import defaultTheme from "@unkey/ui/tailwind.config";
import "tailwindcss/plugin";

module.exports = {
  darkMode: ["class"],
  content: [
    "./pages/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
    "./src/**/*.{ts,tsx}",
    "../../internal/ui/src/**/*.tsx",
    "../../internal/icons/src/**/*.tsx",
  ],
  theme: merge(defaultTheme.theme, {
    /**
     * We need to remove almost all of these and move them into `@unkey/ui`.
     * Especially colors and font sizes need to go
     */
    fontSize: {
      xxs: ["10px", "16px"],
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
      success: {
        DEFAULT: "hsl(var(--success))",
      },

      "amber-2": "var(--amber-2)",
      "amber-3": "var(--amber-3)",
      "amber-4": "var(--amber-4)",
      "amber-6": "var(--amber-6)",
      "amber-11": "var(--amber-11)",

      "red-2": "var(--red-2)",
      "red-3": "var(--red-3)",
      "red-4": "var(--red-4)",
      "red-6": "var(--red-6)",
      "red-11": "var(--red-11)",

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
        "shiny-text": {
          "0%, 90%, 100%": {
            "background-position": "calc(-100% - var(--shiny-width)) 0",
          },
          "30%, 60%": {
            "background-position": "calc(100% + var(--shiny-width)) 0",
          },
        },
        shimmer: {
          "0%": { transform: "translateX(-100%)" },
          "100%": { transform: "translateX(100%)" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "shiny-text": "shiny-text 10s infinite",
        shimmer: "shimmer 1.2s ease-in-out infinite",
      },
      fontFamily: {
        sans: ["var(--font-geist-sans)"],
        mono: ["var(--font-geist-mono)"],
      },
    },
  }),
  plugins: [
    require("tailwindcss-animate"),
    require("@tailwindcss/typography"),
    require("@tailwindcss/aspect-ratio"),
    require("@tailwindcss/container-queries"),
  ],
};

export function merge(obj1, obj2) {
  for (const key in obj2) {
    // biome-ignore lint/suspicious/noPrototypeBuiltins: don't tell me what to do
    if (obj2.hasOwnProperty(key)) {
      if (typeof obj2[key] === "object" && !Array.isArray(obj2[key])) {
        if (!obj1[key]) {
          obj1[key] = {};
        }
        obj1[key] = merge(obj1[key], obj2[key]);
      } else {
        obj1[key] = obj2[key];
      }
    }
  }
  return obj1;
}
