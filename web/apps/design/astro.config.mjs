import mdx from "@astrojs/mdx";
import react from "@astrojs/react";
import tailwind from "@tailwindcss/vite";
import { defineConfig } from "astro/config";
import { previewCodePlugin } from "./src/lib/vite-plugin-preview-code";

export default defineConfig({
  integrations: [react(), mdx()],
  markdown: {
    shikiConfig: {
      themes: { light: "github-light", dark: "github-dark" },
      defaultColor: false,
    },
  },
  vite: {
    plugins: [previewCodePlugin(), tailwind()],
  },
});
