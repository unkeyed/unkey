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

export function ApiUsageExceeded({ workspaceName, used, limit, billingUrl, year }: Props) {
  return (
    <Layout>
      <Preview>{workspaceName} has exceeded the Free plan&apos;s monthly request limit.</Preview>
      <Heading className="font-sans text-3xl font-semibold">
        Your API usage is above the Free plan limit
      </Heading>
      <Text>Hey,</Text>
      <Text>
        <strong>{workspaceName}</strong> has used <strong>{used}</strong> requests this month, above
        the Free plan&apos;s <strong>{limit}</strong>-request limit.
      </Text>
      <Text>
        If this growth is expected, upgrade your workspace to increase its monthly request allowance
        and give your team more room to scale.
      </Text>

      <Section className="text-center py-3">
        <Button href={billingUrl} className="bg-gray-900 text-gray-50 rounded-lg px-7 py-3">
          Review upgrade options
        </Button>
      </Section>

      <Hr />
      <Text>
        Need help choosing a plan? Reply to this email or contact{" "}
        <Link href="mailto:support@unkey.com">support@unkey.com</Link>.
      </Text>
      <Text className="text-xs">
        You&apos;re receiving this because you&apos;re an admin of this workspace.
      </Text>
      <Text className="text-xs">© {year} Unkey</Text>
    </Layout>
  );
}

ApiUsageExceeded.PreviewProps = {
  workspaceName: "Acme Inc",
  used: "182,450",
  limit: "150,000",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default ApiUsageExceeded;
