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
  billingUrl: string;
  year: string;
};

export function ApiUsageRatelimitFollowUp({ workspaceName, used, limit, billingUrl, year }: Props) {
  return (
    <Layout>
      <Preview>
        {workspaceName}&apos;s API traffic is being rate limited and will continue to be reduced to
        one request per hour while it remains over the Free plan limit.
      </Preview>
      <Heading className="font-sans text-3xl font-semibold">We are rate limiting your API</Heading>
      <Text>Hey,</Text>
      <Text>
        <strong>{workspaceName}</strong> is still above its monthly request limit, with{" "}
        <strong>{used}</strong> requests used against a <strong>{limit}</strong>-request allowance.
      </Text>
      <Text>
        Because this workspace remains above the Free plan limit, we have started reducing the rate
        at which it can make API requests.
      </Text>
      <Text>
        We will continue lowering the allowed request rate while the workspace remains above the
        limit, down to one request per hour.
      </Text>
      <Text>
        Upgrade to a paid plan to restore normal request capacity and prevent further reductions.
      </Text>

      <Section className="text-center py-3">
        <Button href={billingUrl} className="bg-gray-900 text-gray-50 rounded-lg px-7 py-3">
          Upgrade your workspace
        </Button>
      </Section>

      <Text>
        If you have already upgraded or believe this limit is incorrect, reply to this email or
        contact <Link href="mailto:support@unkey.com">support@unkey.com</Link>.
      </Text>
      <Hr />
      <Text className="text-xs">
        You&apos;re receiving this because you&apos;re an admin of this workspace.
      </Text>
      <Text className="text-xs">© {year} Unkey</Text>
    </Layout>
  );
}

ApiUsageRatelimitFollowUp.PreviewProps = {
  workspaceName: "Acme Inc",
  used: "240,300",
  limit: "150,000",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default ApiUsageRatelimitFollowUp;
