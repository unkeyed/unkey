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
        md: "72px",
      },
    },
    screens: {
      xxs: "361px",
      xs: "500px",
      sm: "640px",
      md: "840px",
      lg: "960px",
      xl: "1440px",
    },
    extend: {
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
        meteorAngle: "meteorAngle 10s linear infinite",
        meteor: "meteor 20s linear infinite",
      },
      keyframes: {
        meteorAngle: {
          "0%": { tranform: "rotate(300deg) translateX(0)", opacity: "1" },
          "70%": { opacity: "1" },
          "100%": {
            transform: "rotate(300deg) translateX(-400px) ",
            opacity: "0",
          },
        },
        meteor: {
          "0%": { transform: "rotate(270deg) translateX(0)", opacity: ".9" },
          "50%": { opacity: ".4" },
          "100%": {
            transform: "rotate(270deg) translateX(-500px)",
            opacity: "0",
          },
        },
      },
    },
  },
  plugins: [require("tailwindcss-animate"), require("@tailwindcss/typography")],
} satisfies Config;

export default config;
