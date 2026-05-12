/**
 * Static extension registry.
 *
 * P1 ships a hardcoded list so the marketplace UI can be built and reviewed
 * end-to-end without backend dependencies. Replace with a tRPC procedure
 * (or remote manifest) in P5.
 */

export type ExtensionCategory =
  | "logging"
  | "observability"
  | "analytics"
  | "authentication"
  | "webhooks"
  | "alerting"
  | "billing"
  | "ai"
  | "storage"
  | "other";

export type ExtensionType = "native" | "partner" | "community";

export type ExtensionScope = "project" | "workspace";

export type ExtensionPermission = {
  id: string;
  label: string;
  description: string;
};

export type ExtensionVerificationCheck = {
  id: string;
  label: string;
};

export type ExtensionConfigOption = {
  value: string;
  label: string;
  description?: string;
};

/**
 * Field types supported by the install wizard's auto-rendered form.
 *
 * - text/secret/url: single-line inputs (secret is password-masked)
 * - boolean: switch toggle
 * - select: single-choice dropdown (requires `options`)
 * - multiselect: zero-or-more checkbox group (requires `options`)
 * - string-list: free-form list of strings, e.g. exclude-path patterns
 */
export type ExtensionConfigFieldType =
  | "text"
  | "secret"
  | "url"
  | "boolean"
  | "select"
  | "multiselect"
  | "string-list";

export type ExtensionConfigField = {
  id: string;
  label: string;
  type: ExtensionConfigFieldType;
  placeholder?: string;
  helpText?: string;
  required?: boolean;
  /**
   * Logical group label so related fields render together
   * (e.g. "Connection", "Routing", "Filters").
   */
  group?: string;
  /** Required for "select" and "multiselect" types. */
  options?: ExtensionConfigOption[];
  /** Default value applied when the form initializes. */
  defaultValue?: ExtensionConfigValue;
  /** Hide this field unless another field has the given value. */
  dependsOn?: { fieldId: string; equals: ExtensionConfigValue };
};

export type ExtensionConfigValue = string | boolean | string[];

export type ExtensionConfigState = Record<string, ExtensionConfigValue>;

export type ExtensionOAuthConfig = {
  /** Display name used on the Connect button: "Connect with {provider}". */
  provider: string;
  scopes: string[];
};

export type ExtensionLink = {
  docs?: string;
  homepage?: string;
  repo?: string;
};

export type ExtensionVendor = {
  name: string;
  url?: string;
  supportEmail?: string;
};

export type ExtensionChangelogEntry = {
  version: string;
  date: string;
  notes: string;
};

/**
 * Backing implementation for an extension.
 *
 * - "live"    — install/uninstall hits real tRPC + DB. The extension actually
 *               wires up backend resources (e.g. a `log_drains` row).
 * - "preview" — UI-only stub. Installations live in localStorage so we can
 *               demo the manifest, config form, and detail screens without a
 *               backend behind it.
 *
 * The marketplace renders both modes the same way; only the install/uninstall
 * paths and the "Preview" badge differ.
 */
export type ExtensionMode = "live" | "preview";

export type Extension = {
  slug: string;
  name: string;
  tagline: string;
  description: string;
  iconUrl: string;
  vendor: ExtensionVendor;
  categories: ExtensionCategory[];
  type: ExtensionType;
  scope: ExtensionScope;
  /** Whether install actually provisions backend or is a UI-only preview. */
  mode: ExtensionMode;
  verified: boolean;
  installs: number;
  permissions: ExtensionPermission[];
  configFields: ExtensionConfigField[];
  oauth?: ExtensionOAuthConfig;
  verification: ExtensionVerificationCheck[];
  changelog: ExtensionChangelogEntry[];
  links: ExtensionLink;
  /** When true, surface in the curated "Featured" rail. */
  featured?: boolean;
};

export const CATEGORY_LABELS: Record<ExtensionCategory, string> = {
  logging: "Logging",
  observability: "Observability",
  analytics: "Analytics",
  authentication: "Authentication",
  webhooks: "Webhooks",
  alerting: "Alerting",
  billing: "Billing",
  ai: "AI",
  storage: "Storage",
  other: "Other",
};

export const EXTENSION_TYPE_LABELS: Record<ExtensionType, string> = {
  native: "Native",
  partner: "Partner",
  community: "Community",
};

export const EXTENSIONS: Extension[] = [
  {
    slug: "datadog",
    name: "Datadog",
    tagline: "Stream deployment runtime and request logs to Datadog.",
    description:
      "Forward your deployments' runtime stdout/stderr and Sentinel request logs to Datadog as logs and metrics. Build dashboards, set up monitors, and correlate request behavior with the rest of your stack — across every environment.",
    iconUrl: "https://cdn.simpleicons.org/datadog/632CA6",
    vendor: { name: "Unkey", url: "https://unkey.com" },
    categories: ["logging", "observability"],
    type: "native",
    scope: "project",
    mode: "preview",
    verified: true,
    installs: 1284,
    permissions: [
      {
        id: "logs:read",
        label: "Read runtime and request logs",
        description: "Stream deployment runtime and Sentinel request logs to Datadog.",
      },
      {
        id: "metrics:read",
        label: "Read deployment metrics",
        description: "Forward request rate, latency, and error counters per environment.",
      },
    ],
    configFields: [
      {
        id: "apiKey",
        label: "Datadog API key",
        type: "secret",
        required: true,
        helpText: "Found under Organization Settings → API Keys.",
        group: "Connection",
      },
      {
        id: "site",
        label: "Site",
        type: "select",
        required: true,
        group: "Connection",
        defaultValue: "datadoghq.com",
        options: [
          { value: "datadoghq.com", label: "US (datadoghq.com)" },
          { value: "datadoghq.eu", label: "EU (datadoghq.eu)" },
          { value: "us3.datadoghq.com", label: "US3 (us3.datadoghq.com)" },
          { value: "us5.datadoghq.com", label: "US5 (us5.datadoghq.com)" },
        ],
      },
      {
        id: "forwardMetrics",
        label: "Forward metrics",
        type: "boolean",
        group: "Routing",
        defaultValue: true,
        helpText:
          "Send request rate, latency, and error counters as Datadog metrics per environment.",
      },
    ],
    verification: [
      { id: "ingest", label: "Test event reaches Datadog ingest API" },
      { id: "tags", label: "Project tags are present on the test event" },
    ],
    changelog: [
      {
        version: "1.0.0",
        date: "2026-04-12",
        notes: "Initial release with runtime/request log + metrics streaming.",
      },
    ],
    links: { docs: "https://docs.unkey.com/integrations/datadog" },
    featured: true,
  },
  {
    slug: "axiom",
    name: "Axiom",
    tagline: "Ship runtime and request logs to Axiom for fast, queryable storage.",
    description:
      "A native log drain that writes deployment runtime stdout/stderr and Sentinel request logs to an Axiom dataset of your choice. Query with APL, build dashboards, and alert on anomalies across every environment of your project.",
    iconUrl: "",
    vendor: { name: "Unkey", url: "https://unkey.com" },
    categories: ["logging", "observability"],
    type: "native",
    scope: "project",
    // Backed by the real `log_drains` pipeline; gated by the `extensionsLive`
    // flag at the install hook level so preview-mode workspaces still see UI.
    mode: "live",
    verified: true,
    installs: 643,
    permissions: [
      {
        id: "logs:read",
        label: "Read runtime and request logs",
        description: "Stream deployment runtime and Sentinel request logs to Axiom.",
      },
    ],
    configFields: [
      {
        id: "dataset",
        label: "Dataset",
        type: "text",
        placeholder: "unkey-prod",
        required: true,
        group: "Connection",
      },
      {
        id: "endpoint",
        label: "Endpoint",
        type: "url",
        placeholder: "https://api.axiom.co",
        helpText: "Use api.eu.axiom.co for EU.",
        group: "Connection",
      },
      {
        id: "token",
        label: "API token",
        type: "secret",
        required: true,
        placeholder: "xaat-...",
        helpText: "Encrypted in Vault. Never re-displayed; rotate by editing the drain.",
        group: "Connection",
      },
      {
        id: "sources",
        label: "Sources",
        type: "multiselect",
        required: true,
        group: "What to forward",
        defaultValue: ["runtime", "request"],
        options: [
          {
            value: "runtime",
            label: "Runtime logs",
            description: "stdout/stderr from your deployments",
          },
          {
            value: "request",
            label: "Request logs",
            description: "HTTP access logs from Sentinel",
          },
        ],
      },
      {
        id: "environments",
        label: "Environments",
        type: "multiselect",
        required: true,
        group: "What to forward",
        defaultValue: ["production"],
        options: [
          { value: "production", label: "Production" },
          { value: "preview", label: "Preview" },
        ],
      },
      {
        id: "minSeverity",
        label: "Minimum severity",
        type: "select",
        group: "Filters",
        defaultValue: "all",
        helpText: "Drop runtime logs below this level. Reduces provider cost.",
        dependsOn: { fieldId: "sources", equals: "runtime" },
        options: [
          { value: "all", label: "All severities" },
          { value: "debug", label: "Debug and above" },
          { value: "info", label: "Info and above" },
          { value: "warn", label: "Warn and above" },
          { value: "error", label: "Error only" },
        ],
      },
      {
        id: "includeBodies",
        label: "Include request bodies",
        type: "boolean",
        group: "Filters",
        defaultValue: false,
        helpText: "Off by default for privacy. Bodies may contain user data.",
        dependsOn: { fieldId: "sources", equals: "request" },
      },
      {
        id: "excludePaths",
        label: "Exclude paths",
        type: "string-list",
        group: "Filters",
        placeholder: "/health",
        helpText: "Skip request logs whose path matches any of these patterns.",
        dependsOn: { fieldId: "sources", equals: "request" },
      },
    ],
    verification: [
      { id: "ingest", label: "Test event accepted by Axiom ingest endpoint" },
      { id: "dataset", label: "Configured dataset exists and is writable" },
    ],
    changelog: [{ version: "1.0.0", date: "2026-04-08", notes: "Initial release." }],
    links: { docs: "https://docs.unkey.com/integrations/axiom" },
    featured: true,
  },
  {
    slug: "betterstack",
    name: "BetterStack",
    tagline: "Forward deployment logs and page on-call when deploys fail.",
    description:
      "Send your deployment runtime and request logs to BetterStack Logs, and optionally fan failed-deploy or 5xx-spike events out to BetterStack Uptime to page on-call engineers when something goes wrong.",
    iconUrl: "https://cdn.simpleicons.org/betterstack",
    vendor: { name: "BetterStack", url: "https://betterstack.com" },
    categories: ["logging", "alerting"],
    type: "partner",
    scope: "project",
    mode: "preview",
    verified: true,
    installs: 211,
    permissions: [
      {
        id: "logs:read",
        label: "Read runtime and request logs",
        description: "Stream deployment runtime and Sentinel request logs to BetterStack.",
      },
      {
        id: "deployments:read",
        label: "Read deployment events",
        description: "Trigger incidents on failed deployments and rollbacks.",
      },
    ],
    configFields: [
      {
        id: "sourceToken",
        label: "Source token",
        type: "secret",
        required: true,
        helpText: "Generated when you create a Logs source in BetterStack.",
      },
    ],
    verification: [{ id: "ingest", label: "Test event accepted by BetterStack ingest" }],
    changelog: [{ version: "0.9.0", date: "2026-03-21", notes: "Beta release." }],
    links: { docs: "https://betterstack.com/docs/logs/unkey" },
  },
  {
    slug: "slack",
    name: "Slack",
    tagline: "Post deploy, rollback, and incident events to a Slack channel.",
    description:
      "Hook your Slack workspace into Unkey Deploy. Get notified when a deployment succeeds or fails, when a rollback happens, when a custom domain finishes verification, or when Sentinel sees an error spike.",
    iconUrl: "",
    vendor: { name: "Unkey", url: "https://unkey.com" },
    categories: ["alerting", "webhooks"],
    type: "native",
    scope: "workspace",
    mode: "preview",
    verified: true,
    installs: 932,
    permissions: [
      {
        id: "deployments:read",
        label: "Read deployment events",
        description: "Subscribe to deploy success/failure and rollback events.",
      },
      {
        id: "domains:read",
        label: "Read custom domain events",
        description: "Notify when a custom domain finishes verification.",
      },
    ],
    configFields: [
      {
        id: "channel",
        label: "Channel",
        type: "text",
        placeholder: "#alerts",
        required: true,
        helpText: "The Slack channel to post events to.",
      },
    ],
    oauth: {
      provider: "Slack",
      scopes: ["chat:write", "channels:read"],
    },
    verification: [{ id: "auth", label: "Slack OAuth token can post to selected channel" }],
    changelog: [{ version: "1.1.0", date: "2026-04-01", notes: "Add per-event filters." }],
    links: { docs: "https://docs.unkey.com/integrations/slack" },
  },
  {
    slug: "discord",
    name: "Discord",
    tagline: "Pipe deploy and incident events into a Discord channel.",
    description:
      "Wire Unkey Deploy events into a Discord channel via webhook. Configure which events to forward (deploys, rollbacks, custom-domain verification, error spikes) and customize the message template per channel.",
    iconUrl: "https://cdn.simpleicons.org/discord/5865F2",
    vendor: { name: "Community", url: "https://github.com/unkeyed/unkey" },
    categories: ["alerting", "webhooks"],
    type: "community",
    scope: "project",
    mode: "preview",
    verified: false,
    installs: 87,
    permissions: [
      {
        id: "deployments:read",
        label: "Read deployment events",
        description: "Subscribe to deploy and rollback events.",
      },
    ],
    configFields: [
      {
        id: "webhookUrl",
        label: "Webhook URL",
        type: "url",
        placeholder: "https://discord.com/api/webhooks/...",
        required: true,
      },
    ],
    verification: [{ id: "webhook", label: "Discord webhook URL accepts a test message" }],
    changelog: [{ version: "0.2.0", date: "2026-02-14", notes: "Community release." }],
    links: { repo: "https://github.com/unkeyed/extensions/discord" },
  },
  {
    slug: "resend",
    name: "Resend",
    tagline: "Email your team on deploy success, failure, or error spikes.",
    description:
      "Configure email templates and triggers so your team gets a heads up when a deployment fails, a rollback completes, a custom domain finishes verification, or Sentinel sees an error spike on production.",
    iconUrl: "https://cdn.simpleicons.org/resend",
    vendor: { name: "Resend", url: "https://resend.com" },
    categories: ["webhooks", "alerting"],
    type: "partner",
    scope: "project",
    mode: "preview",
    verified: true,
    installs: 318,
    permissions: [
      {
        id: "deployments:read",
        label: "Read deployment events",
        description: "Trigger emails on deploy and rollback events.",
      },
    ],
    configFields: [
      {
        id: "apiKey",
        label: "Resend API key",
        type: "secret",
        required: true,
      },
      {
        id: "from",
        label: "From address",
        type: "text",
        placeholder: "alerts@example.com",
        required: true,
        helpText: "Must be a verified sender in Resend.",
      },
    ],
    verification: [
      { id: "auth", label: "Resend API key authorizes a test send" },
      { id: "sender", label: "Sender domain is verified in Resend" },
    ],
    changelog: [
      {
        version: "1.0.0",
        date: "2026-03-30",
        notes: "Initial templates: deploy failed, deploy succeeded, error spike.",
      },
    ],
    links: { docs: "https://resend.com/docs/integrations/unkey" },
  },
];

export function getExtensionBySlug(slug: string): Extension | undefined {
  return EXTENSIONS.find((extension) => extension.slug === slug);
}

export const ALL_CATEGORIES: ExtensionCategory[] = Object.keys(
  CATEGORY_LABELS,
) as ExtensionCategory[];
