const { withContentlayer } = require("next-contentlayer");
const withBundleAnalyzer = require("@next/bundle-analyzer")();

const securityHeaders = [
  {
    key: "X-Frame-Options",
    value: "SAMEORIGIN",
  },
];
/** @type {import('next').NextConfig} */
const nextConfig = {
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  reactStrictMode: true,
  swcMinify: true,
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: securityHeaders,
      },
    ];
  },
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

let finalNextConfig = nextConfig;
finalNextConfig = withContentlayer(finalNextConfig);

if (process.env.ANALYZE === "true") {
  finalNextConfig = withBundleAnalyzer(finalNextConfig);
}

module.exports = finalNextConfig;
