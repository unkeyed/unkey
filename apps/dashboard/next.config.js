/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  productionBrowserSourceMaps: true,
  // we're open-source anyways
  experimental: {
    esmExternals: "loose",
  },
  webpack: (config) => {
    config.cache = Object.freeze({
      type: "memory",
    });
    return config;
  },
  transpilePackages: ["@unkey/db", "@unkey/resend", "@unkey/vercel", "@unkey/error", "@unkey/id"],
  eslint: {
    // Warning: This allows production builds to successfully complete even if
    // your project has ESLint errors.
    ignoreDuringBuilds: true,
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
  ],
};

module.exports = nextConfig;
