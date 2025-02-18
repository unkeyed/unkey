import unkeyUiTailwindConfig from "@unkey/ui/tailwind.config";
import { createPreset } from "fumadocs-ui/tailwind-plugin";
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
    "./content/**/*.{md,mdx}",
    "./mdx-components.{ts,tsx}",
    "./node_modules/fumadocs-ui/dist/**/*.js",
    "../../internal/ui/src/**/*.tsx",
    "../../internal/icons/src/**/*.tsx",
  ],

  theme: unkeyUiTailwindConfig.theme,
  presets: [createPreset()],
};
