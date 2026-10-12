import { readFileSync } from "node:fs";
import { defineConfig } from "blume";
import { openapi } from "blume/reference";

import {
  selectorsFromModel,
  tabsFromModel,
  toBlumeSidebar,
} from "./scripts/sidebar.mjs";

const readJson = (name: string) =>
  JSON.parse(readFileSync(new URL(name, import.meta.url), "utf8"));

const model = readJson("./sidebar.json");
const redirects = [
  ...readJson("./redirects.json"),
  ...readJson("./alias-redirects.json"),
];
const variables = readJson("./variables.json");

const assistantPrompt = readFileSync(
  new URL("./docs/.mintlify/Assistant.md", import.meta.url),
  "utf8",
);

const agentNotes = [
  "Unkey is two separate products. Compute builds, deploys, and runs apps behind a gateway. API Management issues API keys, enforces rate limits, manages identities and permissions, and reports usage. Say which product a page belongs to; a reader can use either without the other.",
  "Every Unkey API endpoint is an HTTP POST to https://api.unkey.com/v2/{service}.{procedure} with a root key in the Authorization: Bearer header. Root keys are workspace scoped.",
  "Error codes have the form err:{system}:{category}:{specific} and each has a page at /errors/{system}/{category}/{specific}.",
  "The word environment means production or preview in Compute. Rate limiting has four meanings on this site; the glossary lists them.",
].join("\n\n");

export default defineConfig({
  title: "Unkey Docs",
  description:
    "Unkey documentation for two products: Compute, which builds, deploys, and runs your app behind a gateway, and API Management, which issues API keys, enforces rate limits, and reports usage. Platform pages cover the workspace, billing, root keys, the CLI, and error codes shared by both.",
  logo: {
    href: "https://unkey.com",
    image: "/logo.svg",
    text: "",
  },
  theme: {
    accent: "#3756f3",
    action: "#3756f3",
    background: {
      dark: "#040406",
      light: "#ffffff",
    },
    fonts: {
      body: { name: "Inter", weights: [400, 500, 600] },
      display: { name: "Inter", weights: [400, 500, 600] },
      mono: "jetbrains-mono",
    },
    mode: "dark",
    radius: "md",
  },
  variables,
  github: {
    dir: "docsNew/docs",
    owner: "unkeyed",
    repo: "unkey",
  },
  navigation: {
    selectors: selectorsFromModel(model),
    sidebar: {
      display: "group",
      items: toBlumeSidebar(model),
    },
    tabs: tabsFromModel(model),
  },
  footer: {
    links: [
      { href: "/compute", label: "Compute" },
      { href: "/api-management", label: "API management" },
      { href: "https://unkey.com/pricing", label: "Pricing" },
      { href: "https://app.unkey.com", label: "Dashboard" },
      { href: "https://github.com/unkeyed/unkey", label: "GitHub" },
      { href: "https://status.unkey.com", label: "Status" },
      { href: "https://unkey.com/blog", label: "Blog" },
      { href: "https://unkey.com/discord", label: "Discord" },
    ],
    socials: {
      discord: "https://unkey.com/discord",
      github: "https://github.com/unkeyed/unkey",
      linkedin: "https://www.linkedin.com/company/unkeyed",
      x: "https://x.com/unkeydev",
    },
  },
  reference: [
    openapi({
      sources: [
        {
          label: "Platform API",
          route: "/platform/api-reference",
          spec: "./openapi/platform.yaml",
        },
        {
          label: "Compute API",
          route: "/compute/api-reference",
          spec: "./openapi/compute.yaml",
        },
        {
          label: "API management API",
          route: "/api-management/api-reference",
          spec: "./openapi/api-management.yaml",
        },
      ],
    }),
  ],
  redirects,
  ai: {
    assistant: {
      enabled: false,
      instructions: assistantPrompt,
    },
  },
  agents: {
    llmsTxt: {
      details: agentNotes,
    },
  },
  seo: {
    og: { enabled: true },
    robots: true,
    sitemap: true,
    structuredData: true,
  },
  deployment: {
    site: "https://www.unkey.com",
  },
  lastModified: "git",
});
