const { withContentlayer } = require("next-contentlayer");
/** @type {import('next').NextConfig} */

const nextConfig = {
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  reactStrictMode: true,
  swcMinify: true,
  async redirects() {
    return [
      {
        source: "/changelog/:slug",
        destination: "/changelog#:slug", // Matched parameters can be used in the destination
        permanent: true,
      },
    ];
  },
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
      source: "/discord",
      destination: "https://discord.gg/fDbezjbJbD",
    },
    {
      source: "/github",
      destination: "https://github.com/unkeyed/unkey",
    },
    {
      source: "/meet",
      destination: "https://cal.com/team/unkey",
    },
  ],
};

module.exports = withContentlayer(nextConfig);
