import type { Config } from "tailwindcss";

const config = {
  darkMode: ["class"],
  content: [
    "./pages/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
    "./src/**/*.{ts,tsx}",
  ],
  prefix: "",
  theme: {
    container: {
      padding: {
        DEFAULT: "16px",
        sm: "26px",
        md: "30px",
        xl: "72px",
      },
    },
    screens: {
      xxs: "361px",
      xs: "500px",
      sm: "640px",
      md: "840px",
      lg: "960px",
      xl: "960px",
      xxl: "1440px",
    },
    extend: {
      fontSize: {
        xxs: ["10px", "16px"],
      },
      borderWidth: { DEFAULT: "0.75px" },
      opacity: { "02": "0.7 " },
      typography: {},
      borderRadius: {
        "4xl": "2rem",
      },
      fontFamily: {
        sans: ["var(--font-geist-sans)"],
        mono: ["var(--font-geist-mono)"],
      },
      backgroundImage: {
        "gradient-radial": "radial-gradient(var(--tw-gradient-stops))",
      },
      animation: {
        "border-beam": "border-beam calc(var(--duration)*1s) infinite linear",
      },
      keyframes: {
        "border-beam": {
          "100%": {
            "offset-distance": "100%",
          },
        },
      },
    },
  },
  plugins: [require("tailwindcss-animate"), require("@tailwindcss/typography")],
} satisfies Config;

export default config;
