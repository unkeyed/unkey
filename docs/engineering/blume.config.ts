import { type BlumeConfig, defineConfig } from "blume";
import navigation from "./navigation";

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
  title: "Unkey Engineering",
  description: "Internal engineering documentation for Unkey.",
  logo: {
    image: {
      light: "/unkey-black.svg",
      dark: "/unkey-white.svg",
      alt: "Unkey",
    },
    text: "Engineering",
  },
  content: {
    root: ".",
    include: [
      "index.mdx",
      "architecture/**/*.mdx",
      "company/**/*.mdx",
      "contributing/**/*.mdx",
      "infra/**/*.mdx",
    ],
  },
  theme: {
    accent: {
      light: "#8B6914",
      dark: "#F5E6BE",
    },
  },
  navigation,
  github: {
    owner: "unkeyed",
    repo: "unkey",
    dir: "docs/engineering",
  },
  ai: {
    llmsTxt: true,
  },
  deployment: {
    output: "static",
    site: "https://engineering.unkey.com",
  },
  redirects: [
    { from: "/architecture/rfcs/0000-template", to: "/architecture/rfcs/template" },
    { from: "/architecture/rfcs/0001-rbac", to: "/architecture/rfcs/rbac" },
    {
      from: "/architecture/rfcs/0002-github-secret-scanning",
      to: "/architecture/rfcs/github-secret-scanning",
    },
    { from: "/architecture/rfcs/0003-key-shape", to: "/architecture/rfcs/key-shape" },
    { from: "/architecture/rfcs/0004-coss-starter", to: "/architecture/rfcs/coss-starter" },
    { from: "/architecture/rfcs/0005-analytics-api", to: "/architecture/rfcs/analytics-api" },
    { from: "/architecture/rfcs/0006-auth-migration", to: "/architecture/rfcs/auth-migration" },
    {
      from: "/architecture/rfcs/0007-client-file-structure",
      to: "/architecture/rfcs/client-file-structure",
    },
    { from: "/architecture/rfcs/0008-dataplane", to: "/architecture/rfcs/dataplane" },
    { from: "/architecture/rfcs/0009-pricing-updates", to: "/architecture/rfcs/pricing-updates" },
    { from: "/architecture/rfcs/0010-split-monos", to: "/architecture/rfcs/split-monos" },
    {
      from: "/architecture/rfcs/0011-unkey-resource-names",
      to: "/architecture/rfcs/unkey-resource-names",
    },
    { from: "/architecture/rfcs/0012-stricter-linter", to: "/architecture/rfcs/stricter-linter" },
    { from: "/architecture/rfcs/0013-custom-domains", to: "/architecture/rfcs/custom-domains" },
    {
      from: "/architecture/rfcs/0014-frontline-middleware",
      to: "/architecture/rfcs/frontline-middleware",
    },
    {
      from: "/architecture/rfcs/0015-ratelimit-cross-region-counts",
      to: "/architecture/rfcs/ratelimit-cross-region-counts",
    },
    {
      from: "/architecture/rfcs/0016-vault-s3-storage",
      to: "/architecture/rfcs/vault-s3-storage",
    },
    {
      from: "/architecture/rfcs/0014-sentinel-middleware",
      to: "/architecture/rfcs/frontline-middleware",
    },
  ],
});
