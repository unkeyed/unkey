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
  billingUrl: string;
  year: string;
};

export function ComputeBudgetStopped({ workspaceName, usage, budget, billingUrl, year }: Props) {
  return (
    <Layout>
      <Preview>
        {workspaceName} hit its {budget} Compute spend budget, so we stopped its Compute workloads.
      </Preview>
      <Heading className="font-sans text-3xl font-semibold">Compute workloads stopped</Heading>
      <Text>Hey,</Text>
      <Text>
        <strong>{workspaceName}</strong> hit its <strong>{budget}</strong> Compute spend budget for
        the current billing period (<strong>{usage}</strong> used), so we stopped its Compute
        workloads, exactly as you configured.
      </Text>
      <Section>
        <Text className="font-semibold">To start them again, you can:</Text>
        <ul className="pb-4 text-sm">
          <li className="pt-2">Raise or remove the budget</li>
          <li className="pt-2">Turn off &quot;stop workloads&quot; in the budget settings</li>
          <li className="pt-2">Wait for the next billing period</li>
        </ul>
      </Section>

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

ComputeBudgetStopped.PreviewProps = {
  workspaceName: "Acme Inc",
  usage: "$104.20",
  budget: "$100.00",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default ComputeBudgetStopped;
