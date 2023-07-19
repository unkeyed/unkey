/** @type {import('next').NextConfig} */
const { withContentlayer } = require("next-contentlayer");

const nextConfig = {
  pageExtensions: ['tsx', 'mdx' ,'ts', 'js'],
  experimental: {
    serverActions: true,
    esmExternals: "loose",
  },
  eslint: {
    // Warning: This allows production builds to successfully complete even if
    // your project has ESLint errors.
    ignoreDuringBuilds: true,
  },
  // temporary workaround while we see if this works from end to end
  typescript: {
    ignoreBuildErrors: true,
  },
}

module.exports = withContentlayer(nextConfig);