// Renders the emails that the Go control plane sends (by Resend template
// alias) and publishes them to Resend. The components are rendered with literal
// {{{VARIABLE}}} placeholders as props, so Resend substitutes the variables at
// send time; this script is the single place that maps props to variables.
//
// Run manually after changing one of these emails:
//
//   mise exec -- pnpm --dir=web --filter @unkey/resend sync-templates
//
// Requires RESEND_API_KEY. Changes are published immediately, not saved as drafts.
import { render } from "@react-email/render";
import type React from "react";
import { Resend } from "resend";
import ComputeBudgetAlert from "../emails/compute_budget_alert";
import ComputeBudgetStopped from "../emails/compute_budget_stopped";
import FreeTierExceeded from "../emails/free_tier_exceeded";
import FreeTierThrottled from "../emails/free_tier_throttled";
import { SecretScanningKeyDetected } from "../emails/secret_scanning_key_detected";

type TemplateSync = {
  // alias is how svc/ctrl refers to the template; it must not change once a
  // deployed control plane sends with it.
  alias: string;
  name: string;
  subject: string;
  from: string;
  replyTo?: string;
  element: React.ReactElement;
  variables: { key: string; fallbackValue: string }[];
};

const templates: TemplateSync[] = [
  {
    alias: "secret-scanning-key-detected",
    name: "Leaked root key",
    subject: "Unkey root key exposed in public Github repository",
    from: "James from Unkey <james@updates.unkey.com>",
    replyTo: "support@unkey.com",
    element: <SecretScanningKeyDetected date="{{{DATE}}}" source="{{{SOURCE}}}" url="{{{URL}}}" />,
    variables: [
      { key: "DATE", fallbackValue: "" },
      { key: "SOURCE", fallbackValue: "GitHub" },
      { key: "URL", fallbackValue: "https://app.unkey.com" },
    ],
  },
  {
    alias: "compute-budget-alert",
    name: "Compute budget alert",
    subject: "You've used {{{PERCENT}}}% of your spend budget",
    from: "James | Unkey <james@updates.unkey.com>",
    element: (
      <ComputeBudgetAlert
        workspaceName="{{{WORKSPACE_NAME}}}"
        usage="{{{USAGE}}}"
        budget="{{{BUDGET}}}"
        percent="{{{PERCENT}}}"
        billingUrl="{{{BILLING_URL}}}"
        year="{{{YEAR}}}"
      />
    ),
    variables: [
      { key: "WORKSPACE_NAME", fallbackValue: "Your workspace" },
      { key: "USAGE", fallbackValue: "$0" },
      { key: "BUDGET", fallbackValue: "$0" },
      { key: "PERCENT", fallbackValue: "50" },
      { key: "BILLING_URL", fallbackValue: "https://app.unkey.com" },
      { key: "YEAR", fallbackValue: "2026" },
    ],
  },
  {
    alias: "compute-budget-stopped",
    name: "Compute workloads stopped",
    subject: "Compute workloads stopped: budget reached",
    from: "James | Unkey <james@updates.unkey.com>",
    element: (
      <ComputeBudgetStopped
        workspaceName="{{{WORKSPACE_NAME}}}"
        usage="{{{USAGE}}}"
        budget="{{{BUDGET}}}"
        billingUrl="{{{BILLING_URL}}}"
        year="{{{YEAR}}}"
      />
    ),
    variables: [
      { key: "WORKSPACE_NAME", fallbackValue: "Your workspace" },
      { key: "USAGE", fallbackValue: "$0" },
      { key: "BUDGET", fallbackValue: "$0" },
      { key: "BILLING_URL", fallbackValue: "https://app.unkey.com" },
      { key: "YEAR", fallbackValue: "2026" },
    ],
  },
  {
    alias: "free-tier-exceeded",
    name: "Free tier exceeded",
    subject: "You've exceeded the Unkey free tier",
    from: "James | Unkey <james@updates.unkey.com>",
    element: (
      <FreeTierExceeded
        workspaceName="{{{WORKSPACE_NAME}}}"
        used="{{{USED}}}"
        limit="{{{LIMIT}}}"
        requestsPerMinute="{{{REQUESTS_PER_MINUTE}}}"
        throttleDate="{{{THROTTLE_DATE}}}"
        billingUrl="{{{BILLING_URL}}}"
        year="{{{YEAR}}}"
      />
    ),
    variables: [
      { key: "WORKSPACE_NAME", fallbackValue: "Your workspace" },
      { key: "USED", fallbackValue: "0" },
      { key: "LIMIT", fallbackValue: "0" },
      { key: "REQUESTS_PER_MINUTE", fallbackValue: "5" },
      { key: "THROTTLE_DATE", fallbackValue: "in 5 days" },
      { key: "BILLING_URL", fallbackValue: "https://app.unkey.com" },
      { key: "YEAR", fallbackValue: "2026" },
    ],
  },
  {
    alias: "free-tier-throttled",
    name: "Free tier API requests limited",
    subject: "API requests limited: free tier exceeded",
    from: "James | Unkey <james@updates.unkey.com>",
    element: (
      <FreeTierThrottled
        workspaceName="{{{WORKSPACE_NAME}}}"
        limit="{{{LIMIT}}}"
        requestsPerMinute="{{{REQUESTS_PER_MINUTE}}}"
        billingUrl="{{{BILLING_URL}}}"
        year="{{{YEAR}}}"
      />
    ),
    variables: [
      { key: "WORKSPACE_NAME", fallbackValue: "Your workspace" },
      { key: "LIMIT", fallbackValue: "0" },
      { key: "REQUESTS_PER_MINUTE", fallbackValue: "5" },
      { key: "BILLING_URL", fallbackValue: "https://app.unkey.com" },
      { key: "YEAR", fallbackValue: "2026" },
    ],
  },
];

async function main(): Promise<void> {
  const apiKey = process.env.RESEND_API_KEY;
  if (!apiKey) {
    throw new Error("RESEND_API_KEY is not set");
  }
  const resend = new Resend(apiKey);

  for (const template of templates) {
    const html = await render(template.element);
    const text = await render(template.element, { plainText: true });
    const payload = {
      name: template.name,
      subject: template.subject,
      from: template.from,
      replyTo: template.replyTo,
      html,
      text,
      variables: template.variables.map((v) => ({
        key: v.key,
        type: "string" as const,
        fallbackValue: v.fallbackValue,
      })),
    };

    const existing = await resend.templates.get(template.alias);
    if (existing.data) {
      const updated = await resend.templates.update(template.alias, payload);
      if (updated.error) {
        throw new Error(`update ${template.alias}: ${updated.error.message}`);
      }
    } else {
      const created = await resend.templates.create({ ...payload, alias: template.alias });
      if (created.error) {
        throw new Error(`create ${template.alias}: ${created.error.message}`);
      }
    }

    const published = await resend.templates.publish(template.alias);
    if (published.error) {
      throw new Error(`publish ${template.alias}: ${published.error.message}`);
    }

    console.info(`${template.alias}: uploaded and published`);
  }
}

main().catch((err: unknown) => {
  console.error(err);
  process.exitCode = 1;
});
