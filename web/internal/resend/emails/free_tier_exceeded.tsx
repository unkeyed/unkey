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
    <Layout>
      <Preview>{workspaceName} has exceeded the Unkey free tier.</Preview>
      <Heading className="font-sans text-3xl font-semibold">
        You&apos;ve exceeded the free tier
      </Heading>
      <Text>Hey,</Text>
      <Text>
        <strong>{workspaceName}</strong> has used <strong>{used}</strong> requests in a billing
        month, which is over the free tier limit of <strong>{limit}</strong> per month.
      </Text>
      <Text>
        On <strong>{throttleDate}</strong>, we&apos;ll limit API requests for this workspace to{" "}
        <strong>{requestsPerMinute} per minute</strong>. Upgrade before then to keep your current
        limits.
      </Text>

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
