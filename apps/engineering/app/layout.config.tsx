import type { HomeLayoutProps } from "fumadocs-ui/layouts/home";

/**
 * Shared layout configurations
 *
 * you can configure layouts individually from:
 * Home Layout: app/(home)/layout.tsx
 * Docs Layout: app/docs/layout.tsx
 */
export const baseOptions: HomeLayoutProps = {
  nav: {
    title: "Unkey",
  },
  githubUrl: "https://github.com/unkeyed/unkey",
  links: [
    {
      text: "Contributing",
      url: "/contributing",
      active: "nested-url",
    },

    {
      text: "Company",
      url: "/company",
      active: "nested-url",
    },

    {
      text: "Architecture",
      url: "/architecture",
      active: "nested-url",
    },
    {
      text: "RFCs",
      url: "/rfcs",
      active: "nested-url",
    },
    {
      text: "Design",
      url: "/design",
      active: "nested-url",
    },
    {
      text: "GitHub",
      url: "https://github.com/unkeyed/unkey",
    },
  ],
};
