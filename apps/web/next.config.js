const { withContentlayer } = require("next-contentlayer");

/** @type {import('next').NextConfig} */
const nextConfig = {
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  productionBrowserSourceMaps: true, // we're open-source anyways
  experimental: {
    serverActions: true,
    esmExternals: "loose",
  },

  transpilePackages: ["@unkey/db", "@unkey/loops"],
  eslint: {
    // Warning: This allows production builds to successfully complete even if
    // your project has ESLint errors.
    ignoreDuringBuilds: true,
  },
};

const config = withContentlayer(nextConfig);

module.exports = config;
