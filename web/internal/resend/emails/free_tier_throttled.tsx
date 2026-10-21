import { Button } from "@react-email/button";
import { Heading } from "@react-email/heading";
import { Hr } from "@react-email/hr";
import { Link } from "@react-email/link";
import { Preview } from "@react-email/preview";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
// biome-ignore lint/correctness/noUnusedImports: react-email needs this imported
import React from "react";
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
    <Layout>
      <Preview>
        API requests for {workspaceName} are now limited to {requestsPerMinute} per minute.
      </Preview>
      <Heading className="font-sans text-3xl font-semibold">API requests limited</Heading>
      <Text>Hey,</Text>
      <Text>
        <strong>{workspaceName}</strong> went over the free tier limit of <strong>{limit}</strong>{" "}
        requests per month, so we&apos;ve limited its API requests to{" "}
        <strong>{requestsPerMinute} per minute</strong>.
      </Text>
      <Text>Upgrade to a paid plan to remove the limit right away.</Text>

      <Section className="text-center py-3">
        <Button href={billingUrl} className="bg-gray-900 text-gray-50 rounded-lg px-7 py-3">
          Upgrade
        </Button>
      </Section>

      <Hr />
      <Text>
        Need help? Please reach out to{" "}
        <Link href="mailto:support@unkey.com">support@unkey.com</Link> or just reply to this email.
      </Text>
      <Text className="text-xs">© {year} Unkey</Text>
    </Layout>
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
