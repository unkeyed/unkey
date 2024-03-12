/** @type {import('next').NextConfig} */

const APP_URL = process.env.APP_URL ?? "https://app.unkey.dev";

const nextConfig = {
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  rewrites: () => [
    {
      source: "/docs",
      destination: "https://unkey.mintlify.dev/docs",
    },
    {
      source: "/docs/:match*",
      destination: "https://unkey.mintlify.dev/docs/:match*",
    },
    {
      source: "/engineering",
      destination: "https://unkey-engineering.mintlify.dev/engineering",
    },
    {
      source: "/engineering/:match*",
      destination: "https://unkey-engineering.mintlify.dev/engineering/:match*",
    },
    {
      source: "/:path*",
      destination: "/:path*",
    },
    {
      source: "/blog",
      destination: `${APP_URL}/blog`,
    },
    {
      source: "/app/:path*",
      destination: `${APP_URL}/app/:path*`,
    },
  ],
};

export default nextConfig;
