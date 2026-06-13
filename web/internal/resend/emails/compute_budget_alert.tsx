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

// This email is not sent from the dashboard. It is rendered with Resend
// {{{VARIABLE}}} placeholders as props and uploaded as a Resend template by
// scripts/sync-templates.tsx; the Go control plane sends it by template alias.
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
    <Layout>
      <Preview>
        {workspaceName} has used {usage} of its {budget} Compute spend budget. Nothing has been
        stopped.
      </Preview>
      <Heading className="font-sans text-3xl font-semibold">
        You&apos;ve used {percent}% of your spend budget
      </Heading>
      <Text>Hey,</Text>
      <Text>
        <strong>{workspaceName}</strong> has used <strong>{usage}</strong> of its{" "}
        <strong>{budget}</strong> Compute spend budget for this billing period. Nothing has been
        stopped, this is just a heads up.
      </Text>
      <Text>
        If that&apos;s expected, you can ignore this email. Otherwise you can review your usage or
        adjust the budget in your billing settings.
      </Text>

      <Section className="text-center py-3">
        <Button href={billingUrl} className="bg-gray-900 text-gray-50 rounded-lg px-7 py-3">
          Manage billing
        </Button>
      </Section>

      <Hr />
      <Text>
        Need help? Please reach out to{" "}
        <Link href="mailto:support@unkey.com">support@unkey.com</Link> or just reply to this email.
      </Text>

      <Text className="text-xs">
        You&apos;re receiving this because you set a Compute spend budget for this workspace.
      </Text>
      <Text className="text-xs">© {year} Unkey</Text>
    </Layout>
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
