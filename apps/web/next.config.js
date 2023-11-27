const { withContentlayer } = require("next-contentlayer");
const { withSentryConfig } = require("@sentry/nextjs");

/** @type {import('next').NextConfig} */
let nextConfig = {
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  productionBrowserSourceMaps: true, // we're open-source anyways
  experimental: {
    esmExternals: "loose",
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

nextConfig = withSentryConfig(
  nextConfig,
  {
    // For all available options, see:
    // https://github.com/getsentry/sentry-webpack-plugin#options

    // Suppresses source map uploading logs during build
    silent: true,
    org: process.env.SENTRY_ORG,
    project: process.env.SENTRY_PROJECT,
  },
  {
    // For all available options, see:
    // https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/

    // Upload a larger set of source maps for prettier stack traces (increases build time)
    widenClientFileUpload: true,

    // Transpiles SDK to be compatible with IE11 (increases bundle size)
    transpileClientSDK: true,

    // Routes browser requests to Sentry through a Next.js rewrite to circumvent ad-blockers (increases server load)
    tunnelRoute: "/monitoring",

    // Hides source maps from generated client bundles
    hideSourceMaps: true,

    // Automatically tree-shake Sentry logger statements to reduce bundle size
    disableLogger: true,
  },
);

module.exports = nextConfig;
