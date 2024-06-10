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
    extend: {
      container: {
        padding: {
          DEFAULT: "24px",
          sm: "32px",
          md: "32px",
          xl: "32px",
        },
        center: true,
        screens: {
          sm: "640px",
          md: "768px",
          lg: "1024px",
          // xl: "1280px",
          // "2xl": "1280px",
          xl: "1200px",
          "2xl": "1200px",
        },
      },
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
        meteorAngle: "meteorAngle 10s linear infinite",
        meteor: "meteor 20s linear infinite",
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "button-shine": "shine .6s linear forwards",
        "logo-cloud": "logo-cloud 30s linear infinite",
        "fade-in": "fade-in 1s ease-out forwards",
        "fade-in-right": "fade-in-right 1s ease-out forwards",
        "fade-in-left": "fade-in-left 1s ease-out forwards",
        "fade-in-up": "fade-in-up 1s ease-out forwards",
        "fade-in-down": "fade-in-down 1s ease-out forwards",
      },
      keyframes: {
        shine: {
          "0%": {
            backgroundPosition: "0 0",
            opacity: "0",
          },
          "1%": {
            backgroundPosition: "0 0",
            opacity: "1",
          },
          "80%": {
            backgroundPosition: "180% 0",
            opacity: "1",
          },
          "85%": {
            opacity: "0",
          },
        },
        "border-beam": {
          "100%": {
            "offset-distance": "100%",
          },
        },
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
        "fade-in": {
          "0%": { opacity: "0" },
          "100%": { opacity: "1" },
        },
        "fade-in-left": {
          "0%": { opacity: "0", transform: "translateX(var(--fade-in-left-tx, 1rem))" },
          "100%": { opacity: "1", transform: "translateX(0)" },
        },
        "fade-in-right": {
          "0%": { opacity: "0", transform: "translateX(var(--fade-in-right-tx, -1rem))" },
          "100%": { opacity: "1", transform: "translateX(0)" },
        },
        "fade-in-up": {
          "0%": { opacity: "0", transform: "translateY(var(--fade-in-up-ty, 1rem))" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        "fade-in-down": {
          "0%": { opacity: "0", transform: "translateY(var(--fade-in-down-ty, -1rem))" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        "logo-cloud": {
          from: { transform: "translateX(0)" },
          to: { transform: "translateX(calc(-100% - 4rem))" },
        },
        meteorAngle: {
          "0%": { tranform: "rotate(300deg) translateX(0)", opacity: "1" },
          "70%": { opacity: "1" },
          "100%": {
            transform: "rotate(300deg) translateX(-400px) ",
            opacity: "0",
          },
        },
        meteor: {
          "0%": { transform: "rotate(270deg) translateX(0)", opacity: "0" },
          "5%": { opacity: ".9" },
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
