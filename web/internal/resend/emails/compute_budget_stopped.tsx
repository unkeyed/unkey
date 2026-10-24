import { Layout } from "../src/components/layout";

export type Props = {
  workspaceName: string;
  usage: string;
  budget: string;
  billingUrl: string;
  year: string;
};

export function ComputeBudgetStopped({ workspaceName, usage, budget, billingUrl, year }: Props) {
  return (
    <Layout
      title="Compute workloads stopped"
      preview={`${workspaceName} hit its ${budget} Compute spend budget, so we stopped its Compute workloads.`}
      body={[
        `${workspaceName} hit its ${budget} Compute spend budget for this billing period (${usage} used). We stopped its Compute workloads as configured in your budget settings.`,
      ]}
      action={{ label: "Manage billing", href: billingUrl }}
      sections={[
        {
          type: "list",
          title: "To start workloads again",
          items: [
            "Raise or remove the budget.",
            'Turn off "stop workloads" in the budget settings.',
            "Wait for the next billing period.",
          ],
        },
      ]}
      reason="You're receiving this because you set a Compute spend budget for this workspace."
      year={year}
    />
  );
}

ComputeBudgetStopped.PreviewProps = {
  workspaceName: "Acme Inc",
  usage: "$104.20",
  budget: "$100.00",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default ComputeBudgetStopped;
