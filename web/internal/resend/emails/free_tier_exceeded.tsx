import { Layout } from "../src/components/layout";

export type Props = {
  workspaceName: string;
  used: string;
  limit: string;
  requestsPerMinute: string;
  throttleDate: string;
  billingUrl: string;
  year: string;
};

export function FreeTierExceeded({
  workspaceName,
  used,
  limit,
  requestsPerMinute,
  throttleDate,
  billingUrl,
  year,
}: Props) {
  return (
    <Layout
      title="You've exceeded the free tier"
      preview={`${workspaceName} has exceeded the Unkey free tier. Upgrade before ${throttleDate} to keep your current limits.`}
      body={[
        `${workspaceName} has used ${used} requests this billing month, exceeding the free tier limit of ${limit} per month.`,
        `On ${throttleDate}, we'll limit API requests for this workspace to ${requestsPerMinute} per minute. Upgrade before then to keep your current limits.`,
      ]}
      action={{ label: "Upgrade plan", href: billingUrl }}
      year={year}
    />
  );
}

FreeTierExceeded.PreviewProps = {
  workspaceName: "Acme Inc",
  used: "182,431",
  limit: "150,000",
  requestsPerMinute: "5",
  throttleDate: "October 25, 2026",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default FreeTierExceeded;
