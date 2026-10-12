import { defineComponents } from "blume";

export default defineComponents({
  layout: {
    Footer: "./components/Footer.astro",
    PageFooter: "./components/ProductTabs.astro",
  },
  mdx: {
    DashboardScreenshot: "./components/DashboardScreenshot.astro",
    ProductLink: "./components/ProductLink.astro",
  },
});
