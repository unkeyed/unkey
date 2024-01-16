const { withContentlayer } = require("next-contentlayer");
const { withHydrationOverlay } = require("@builder.io/react-hydration-overlay/next");

/** @type {import('next').NextConfig} */
let nextConfig = {
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  productionBrowserSourceMaps: true, // we're open-source anyways
  experimental: {
    esmExternals: "loose",
  },
  webpack: (config) => {
    config.cache = Object.freeze({
      type: "memory",
    });
    return config;
  },

  transpilePackages: ["@unkey/db", "@unkey/resend", "@unkey/vercel", "@unkey/result", "@unkey/id"],
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
  ],
};

nextConfig = withContentlayer(nextConfig);
if (process.env.NODE_ENV !== "production") {
  nextConfig = withHydrationOverlay({ appRootselector: "main" })(nextConfig);
}
module.exports = nextConfig;
