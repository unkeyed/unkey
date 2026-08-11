import { type BlumeConfig, defineConfig } from "blume";

const ampOrbViteServer: NonNullable<BlumeConfig["integrations"]>[number] = {
  name: "unkey:amp-orb-vite-server",
  hooks: {
    "astro:config:setup": ({ updateConfig }) => {
      if (!process.env.AMP_ORB) {
        return;
      }

      updateConfig({
        vite: {
          server: {
            allowedHosts: [".e2b.app", ".onamp.dev"],
            cors: {
              origin: [
                /^https?:\/\/(?:(?:[^:]+\.)?localhost|127\.0\.0\.1|\[::1\])(?::\d+)?$/,
                /^https:\/\/[a-z0-9-]+\.(?:e2b\.app|onamp\.dev)$/,
              ],
            },
          },
        },
      });
    },
  },
};

export default defineConfig({
  integrations: [ampOrbViteServer],
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
