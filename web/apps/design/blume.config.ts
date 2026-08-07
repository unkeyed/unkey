import { defineConfig } from "blume";

export default defineConfig({
  title: "Unkey Design",
  description: "Design system guidance for building consistent Unkey interfaces.",
  logo: {
    image: "/unkey-logo.svg",
    text: "Unkey Design Docs",
  },
  content: {
    root: "docs",
  },
  feedback: false,
  examples: {
    source: "examples",
    css: "preview.css",
  },
  theme: {
    accent: "blue",
    radius: "md",
    mode: "system",
  },
  search: {
    provider: "orama",
  },
  markdown: {
    code: {
      icons: true,
      wrap: false,
    },
  },
  ai: {
    llmsTxt: true,
  },
  seo: {
    og: {
      enabled: false,
    },
  },
});
