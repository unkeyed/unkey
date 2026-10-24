import { Layout } from "../src/components/layout";

export type Props = {
  workspaceName: string;
  limit: string;
  requestsPerMinute: string;
  billingUrl: string;
  year: string;
};

export function FreeTierThrottled({
  workspaceName,
  limit,
  requestsPerMinute,
  billingUrl,
  year,
}: Props) {
  return (
    <Layout
      title="API requests limited"
      preview={`API requests for ${workspaceName} are now limited to ${requestsPerMinute} per minute.`}
      body={[
        `${workspaceName} exceeded the free tier limit of ${limit} requests per month. Its API requests are now limited to ${requestsPerMinute} per minute.`,
        "Upgrade to a paid plan to remove this limit right away.",
      ]}
      action={{ label: "Upgrade plan", href: billingUrl }}
      year={year}
    />
  );
}

FreeTierThrottled.PreviewProps = {
  workspaceName: "Acme Inc",
  limit: "150,000",
  requestsPerMinute: "5",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default FreeTierThrottled;
