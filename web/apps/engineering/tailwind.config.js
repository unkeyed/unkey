import { createPreset } from "fumadocs-ui/tailwind-plugin";

/** @type {import('tailwindcss').Config} */
module.exports = {
  // In Tailwind v4, most configuration is done in CSS via @theme blocks
  // in global.css. This config file is kept minimal for plugin compatibility.
  darkMode: ["class"],
  presets: [createPreset()],
};
