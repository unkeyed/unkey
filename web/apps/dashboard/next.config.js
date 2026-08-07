const isDev = process.env.NODE_ENV === "development";
// Not `!isDev`: NODE_ENV is also "test", which is neither, and HSTS must only
// ever be sent by a real production build.
const isProd = process.env.NODE_ENV === "production";

// The Vercel toolbar is injected on preview deployments and rendered manually
// in development (app/layout.tsx); production never loads it, so production
// must not whitelist its origins.
const allowVercelToolbar = isDev || process.env.VERCEL_ENV === "preview";

// Enforced policy: only directives that cannot break the app. These are
// deliberately NOT repeated in the report-only policy below — keeping each
// directive in exactly one header avoids duplicate console reports per
// violation. `frame-ancestors` mirrors X-Frame-Options; the spec forbids it
// in report-only policies anyway.
const cspEnforced = ["object-src 'none'", "base-uri 'self'", "frame-ancestors 'self'"].join("; ");

const scriptSrc = [
  "'self'",
  // Next.js emits inline bootstrap scripts. Tighten to nonces before
  // enforcing (requires wiring a nonce through proxy.ts and forces dynamic
  // rendering).
  "'unsafe-inline'",
  // Webpack/react-refresh eval only exists in dev; keeping it out of the
  // production policy means any real eval usage shows up as a report-only
  // violation instead of being silently allowed.
  ...(isDev ? ["'unsafe-eval'"] : []),
  ...(allowVercelToolbar ? ["https://vercel.live"] : []),
].join(" ");

const connectSrc = [
  "'self'",
  // Sentry is tunneled through /monitoring (same-origin), but session replay
  // uploads can go straight to the ingest host. Region-specific and
  // intentionally hardcoded (this file has no DSN to derive it from): if the
  // Sentry org ever migrates regions, update this alongside the DSNs in the
  // three sentry.*.config/instrumentation files — drift shows up as
  // connect-src report-only violations.
  "https://*.ingest.us.sentry.io",
  // The preview toolbar talks to vercel.live and Pusher websockets.
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
  // Member avatars come from the auth provider and can live on arbitrary
  // https hosts (Google, GitHub, ...).
  "img-src 'self' blob: data: https:",
  `font-src ${fontSrc}`,
  `connect-src ${connectSrc}`,
  // Sentry session replay runs in a blob: worker.
  "worker-src 'self' blob:",
  ...(allowVercelToolbar ? ["frame-src https://vercel.live"] : []),
  "form-action 'self'",
].join("; ");

const securityHeaders = [
  {
    key: "X-Frame-Options",
    value: "SAMEORIGIN",
  },
  // HSTS only in production builds so a local https session (e.g.
  // *.unkey.local via `mise run tunnel`) doesn't pin dev hosts to https for
  // two years. `preload` is deliberately absent: the dashboard serves from a
  // subdomain, and preload-list submission is an apex-domain (unkey.com)
  // decision with a months-long exit path. Note for self-hosters:
  // includeSubDomains pins every subdomain of the deployment host to https
  // for the max-age — remove it if plain-http services live under the
  // dashboard's hostname.
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
  productionBrowserSourceMaps: true,
  // we're open-source anyways

  poweredByHeader: false,
  webpack: (config) => {
    config.cache = Object.freeze({
      type: "memory",
    });
    return config;
  },
  transpilePackages: ["@unkey/db", "@unkey/resend", "@unkey/error", "@unkey/id"],
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

  // For all available options, see:
  // https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/

  // Upload a larger set of source maps for prettier stack traces (increases build time)
  widenClientFileUpload: true,

  // Route browser requests to Sentry through a Next.js rewrite to circumvent ad-blockers.
  // This can increase your server load as well as your hosting bill.
  // Note: Check that the configured route will not match with your Next.js middleware, otherwise reporting of client-
  // side errors will fail.
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
