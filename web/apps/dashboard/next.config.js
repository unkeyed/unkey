const isDev = process.env.NODE_ENV === "development";
const isProd = process.env.NODE_ENV === "production";

// The Vercel toolbar is injected on preview deployments and rendered manually
// in development (app/layout.tsx); production never loads it, so production
// must not whitelist its origins.
const allowVercelToolbar = isDev || process.env.VERCEL_ENV === "preview";

function shouldUploadSentrySourceMaps() {
  const raw = process.env.SENTRY_UPLOAD_SOURCEMAPS;
  if (raw === undefined || raw.trim() === "") {
    return process.env.VERCEL_ENV === "production";
  }

  const flag = raw.trim().toLowerCase();
  if (flag === "true" || flag === "1") {
    return true;
  }
  if (flag === "false" || flag === "0") {
    return false;
  }

  throw new Error(
    `Invalid SENTRY_UPLOAD_SOURCEMAPS=${JSON.stringify(raw)}. Use true, false, 1, 0, or leave it unset.`,
  );
}

const uploadSentrySourceMaps = shouldUploadSentrySourceMaps();

const sentryReleaseName =
  process.env.SENTRY_RELEASE || process.env.VERCEL_GIT_COMMIT_SHA || "unkey-dashboard";


const cspEnforced = ["object-src 'none'", "base-uri 'self'", "frame-ancestors 'self'"].join("; ");

const scriptSrc = [
  "'self'",
  "'unsafe-inline'",
  ...(isDev ? ["'unsafe-eval'"] : []),
  ...(allowVercelToolbar ? ["https://vercel.live"] : []),
].join(" ");

const connectSrc = [
  "'self'",
  "https://*.ingest.us.sentry.io",
  ...(allowVercelToolbar ? ["https://vercel.live", "wss://*.pusher.com"] : []),
].join(" ");

// Tailwind/inline style attributes and the chart theme <style> tag; the
// preview toolbar additionally loads its stylesheet and fonts from
// vercel.live, and without these allowances previews would flood the console
// with self-inflicted violations.
const styleSrc = [
  "'self'",
  "'unsafe-inline'",
  ...(allowVercelToolbar ? ["https://vercel.live"] : []),
].join(" ");

const fontSrc = [
  "'self'",
  "data:",
  ...(allowVercelToolbar ? ["https://vercel.live/fonts"] : []),
].join(" ");

// Full policy runs in report-only mode: violations are reported to the
// browser console (devtools) but nothing is blocked. There is no report-uri:
// collection is deliberately console-only, so exercise the app in dev/preview
// and watch for violations before folding these directives into the enforced
// Content-Security-Policy header. The already-enforced directives (object-src,
// base-uri, frame-ancestors) are deliberately absent — see cspEnforced above.
const cspReportOnly = [
  "default-src 'self'",
  `script-src ${scriptSrc}`,
  `style-src ${styleSrc}`,
  "img-src 'self' blob: data: https:",
  `font-src ${fontSrc}`,
  `connect-src ${connectSrc}`,
  "worker-src 'self' blob:",
  ...(allowVercelToolbar ? ["frame-src https://vercel.live"] : []),
  "form-action 'self'",
].join("; ");

const securityHeaders = [
  {
    key: "X-Frame-Options",
    value: "SAMEORIGIN",
  },

  ...(isProd
    ? [
        {
          key: "Strict-Transport-Security",
          value: "max-age=63072000; includeSubDomains",
        },
      ]
    : []),
  {
    key: "X-Content-Type-Options",
    value: "nosniff",
  },
  {
    key: "Referrer-Policy",
    value: "strict-origin-when-cross-origin",
  },
  {
    key: "Permissions-Policy",
    value: "camera=(), microphone=(), geolocation=(), browsing-topics=()",
  },
  {
    key: "Content-Security-Policy",
    value: cspEnforced,
  },
  {
    key: "Content-Security-Policy-Report-Only",
    value: cspReportOnly,
  },
];

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  reactStrictMode: true,
  typedRoutes: true,
  allowedDevOrigins: process.env.AMP_ORB ? ["*.onamp.dev", "*.e2b.app"] : undefined,
  pageExtensions: ["tsx", "mdx", "ts", "js"],
  ...(uploadSentrySourceMaps ? {} : { productionBrowserSourceMaps: false }),

  poweredByHeader: false,
  webpack: (config) => {
    config.cache = Object.freeze({
      type: "memory",
    });
    return config;
  },
  transpilePackages: ["@unkey/db", "@unkey/resend", "@unkey/error", "@unkey/id"],
  async redirects() {
    return [
      {
        source: "/:workspaceSlug/projects/:projectId/apps/:appId/sentinel-policies",
        destination: "/:workspaceSlug/projects/:projectId/apps/:appId/policies",
        permanent: true,
      },
    ];
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: securityHeaders,
      },
    ];
  },
};

module.exports = nextConfig;

// Injected content via Sentry wizard below

const { withSentryConfig } = require("@sentry/nextjs");

module.exports = withSentryConfig(module.exports, {
  // For all available options, see:
  // https://www.npmjs.com/package/@sentry/webpack-plugin#options

  org: "unkey-fw",
  project: "unkey-dashboard",

  // Only print logs for uploading source maps in CI
  silent: !process.env.CI,
  useRunAfterProductionCompileHook: true,
  widenClientFileUpload: true,

  ...(uploadSentrySourceMaps
    ? {
        // A thrown upload error fails the build before client maps are deleted.
        errorHandler(error) {
          throw error;
        },
        release: {
          name: sentryReleaseName,
        },
      }
    : {
        sourcemaps: {
          disable: true,
        },
        release: {
          name: sentryReleaseName,
          create: false,
          finalize: false,
          setCommits: false,
          deploy: false,
        },
      }),
  tunnelRoute: "/monitoring",

  webpack: {
    // Enables automatic instrumentation of Vercel Cron Monitors. (Does not yet work with App Router route handlers.)
    // See the following for more information:
    // https://docs.sentry.io/product/crons/
    // https://vercel.com/docs/cron-jobs
    automaticVercelMonitors: true,

    // Tree-shaking options for reducing bundle size
    treeshake: {
      // Automatically tree-shake Sentry logger statements to reduce bundle size
      removeDebugLogging: true,
    },
  },
});

const withVercelToolbar = require("@vercel/toolbar/plugins/next")();
module.exports = withVercelToolbar(module.exports);
