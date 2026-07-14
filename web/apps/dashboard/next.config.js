const isDev = process.env.NODE_ENV === "development";

// The Vercel toolbar is injected on preview deployments and rendered manually
// in development (app/layout.tsx); production never loads it, so production
// must not whitelist its origins.
const allowVercelToolbar = isDev || process.env.VERCEL_ENV === "preview";

// Single literal so both policies always report to the same endpoint.
const cspReportDirective = "report-uri /api/csp-reports";

// Enforced policy: only directives that cannot break the app. These are
// deliberately NOT repeated in the report-only policy — a directive present
// in both headers makes the browser send two reports per violation,
// double-counting against the forward budget and duplicating Sentry events.
// (One residual overlap is accepted: an <object> violation also trips the
// report-only default-src fallback, so it double-reports. Nothing in the app
// embeds objects, so any such report is signal, not noise.)
// `frame-ancestors` mirrors X-Frame-Options; the spec forbids it in
// report-only policies anyway.
const cspEnforced = [
  "object-src 'none'",
  "base-uri 'self'",
  "frame-ancestors 'self'",
  cspReportDirective,
].join("; ");

const scriptSrc = [
  "'self'",
  // Next.js emits inline bootstrap scripts. Tighten to nonces before
  // enforcing (requires wiring a nonce through proxy.ts and forces dynamic
  // rendering).
  "'unsafe-inline'",
  // Webpack/react-refresh eval only exists in dev; keeping it out of the
  // production policy means any real eval usage shows up in the report
  // stream instead of being silently allowed.
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
  // connect-src violations in the report stream.
  "https://*.ingest.us.sentry.io",
  // The preview toolbar talks to vercel.live and Pusher websockets.
  ...(allowVercelToolbar ? ["https://vercel.live", "wss://*.pusher.com"] : []),
].join(" ");

// Tailwind/inline style attributes and the chart theme <style> tag; the
// preview toolbar additionally loads its stylesheet and fonts from
// vercel.live, and without these allowances previews would flood the report
// stream with self-inflicted violations.
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

// Full policy runs in report-only mode: violations are POSTed to
// /api/csp-reports (which scrubs URLs and forwards to Sentry when the SDK is
// enabled) but nothing is blocked. Once the report stream is quiet, fold
// these directives into the enforced Content-Security-Policy header. The
// already-enforced directives (object-src, base-uri, frame-ancestors) are
// deliberately absent — see cspEnforced above.
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
  cspReportDirective,
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
  ...(process.env.NODE_ENV === "production"
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
