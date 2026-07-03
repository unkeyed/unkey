// Renders the emails that the Go control plane sends (by Resend template
// alias) and uploads them to Resend. The components are rendered with literal
// {{{VARIABLE}}} placeholders as props, so Resend substitutes the variables at
// send time; this script is the single place that maps props to variables.
//
// Run manually after changing one of these emails:
//
//   RESEND_API_KEY=... pnpm sync-templates             # upload as draft versions
//   RESEND_API_KEY=... pnpm sync-templates --publish   # upload and publish
//
// A draft version is invisible to sends: the Go side keeps sending the last
// published version until --publish (or the Resend dashboard) publishes it.
import { render } from "@react-email/render";
import type React from "react";
import { Resend } from "resend";
import ComputeBudgetAlert from "../emails/compute_budget_alert";
import ComputeBudgetStopped from "../emails/compute_budget_stopped";

type TemplateSync = {
  // alias is how svc/ctrl refers to the template; it must not change once a
  // deployed control plane sends with it.
  alias: string;
  name: string;
  subject: string;
  from: string;
  element: React.ReactElement;
  variables: { key: string; fallbackValue: string }[];
};

const templates: TemplateSync[] = [
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
];

async function main(): Promise<void> {
  const apiKey = process.env.RESEND_API_KEY;
  if (!apiKey) {
    throw new Error("RESEND_API_KEY is not set");
  }
  const publish = process.argv.includes("--publish");
  const resend = new Resend(apiKey);

  for (const template of templates) {
    const html = await render(template.element);
    const text = await render(template.element, { plainText: true });
    const payload = {
      name: template.name,
      subject: template.subject,
      from: template.from,
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

    if (publish) {
      const published = await resend.templates.publish(template.alias);
      if (published.error) {
        throw new Error(`publish ${template.alias}: ${published.error.message}`);
      }
    }

    console.info(`${template.alias}: ${publish ? "uploaded and published" : "uploaded as draft"}`);
  }
}

main().catch((err: unknown) => {
  console.error(err);
  process.exitCode = 1;
});
