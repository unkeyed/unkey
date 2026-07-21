import type { ReactElement } from "react";
import ApiUsageExceeded from "../emails/api_usage_exceeded";
import ApiUsageRatelimitFollowUp from "../emails/api_usage_ratelimit_follow_up";
import ComputeBudgetAlert from "../emails/compute_budget_alert";
import ComputeBudgetStopped from "../emails/compute_budget_stopped";

export type TemplateSync = {
  // alias is how svc/ctrl refers to the template; it must not change once a
  // deployed control plane sends with it.
  alias: string;
  name: string;
  subject: string;
  from: string;
  element: ReactElement;
  variables: { key: string; fallbackValue: string }[];
};

const apiUsageVariables: TemplateSync["variables"] = [
  { key: "WORKSPACE_NAME", fallbackValue: "Your workspace" },
  { key: "USED", fallbackValue: "150,000" },
  { key: "LIMIT", fallbackValue: "150,000" },
  { key: "BILLING_URL", fallbackValue: "https://app.unkey.com" },
  { key: "YEAR", fallbackValue: "2026" },
];

export const templates: TemplateSync[] = [
  {
    alias: "api-usage-exceeded",
    name: "API usage exceeded",
    subject: "Your API usage is above the Free plan limit",
    from: "James | Unkey <james@updates.unkey.com>",
    element: (
      <ApiUsageExceeded
        workspaceName="{{{WORKSPACE_NAME}}}"
        used="{{{USED}}}"
        limit="{{{LIMIT}}}"
        billingUrl="{{{BILLING_URL}}}"
        year="{{{YEAR}}}"
      />
    ),
    variables: apiUsageVariables,
  },
  {
    alias: "api-usage-ratelimit-follow-up",
    name: "API usage rate limit follow-up",
    subject: "We are rate limiting your API",
    from: "James | Unkey <james@updates.unkey.com>",
    element: (
      <ApiUsageRatelimitFollowUp
        workspaceName="{{{WORKSPACE_NAME}}}"
        used="{{{USED}}}"
        limit="{{{LIMIT}}}"
        billingUrl="{{{BILLING_URL}}}"
        year="{{{YEAR}}}"
      />
    ),
    variables: apiUsageVariables,
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
];
