import { createPreset } from "fumadocs-ui/tailwind-plugin";

/** @type {import('tailwindcss').Config} */
module.exports = {
  // Fumadocs UI v14.4.0 preset (compatible with Tailwind v4)
  // Most configuration is in CSS via @theme blocks in global.css
  darkMode: ["class"],
  presets: [createPreset()],
};
