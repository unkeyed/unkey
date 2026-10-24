import { Layout } from "../src/components/layout";

export type Props = {
  workspaceName: string;
  usage: string;
  budget: string;
  percent: string;
  billingUrl: string;
  year: string;
};

export function ComputeBudgetAlert({
  workspaceName,
  usage,
  budget,
  percent,
  billingUrl,
  year,
}: Props) {
  return (
    <Layout
      title={`You've used ${percent}% of your spend budget`}
      preview={`${workspaceName} has used ${usage} of its ${budget} Compute spend budget. Nothing has been stopped.`}
      body={[
        `${workspaceName} has used ${usage} of its ${budget} Compute spend budget for this billing period. Nothing has been stopped.`,
        "If this is expected, no action is needed. Otherwise, review your usage or adjust the budget in your billing settings.",
      ]}
      action={{ label: "Manage billing", href: billingUrl }}
      reason="You're receiving this because you set a Compute spend budget for this workspace."
      year={year}
    />
  );
}

ComputeBudgetAlert.PreviewProps = {
  workspaceName: "Acme Inc",
  usage: "$81.40",
  budget: "$100.00",
  percent: "80",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default ComputeBudgetAlert;
