import { Button } from "@react-email/button";
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
  used: string;
  limit: string;
  billingUrl: string;
  year: string;
};

export function FreeApiQuotaWarning({ workspaceName, used, limit, billingUrl, year }: Props) {
  return (
    <Layout>
      <Preview>
        {workspaceName} is over its free API limit. Nothing is being rate limited yet.
      </Preview>
      <Text>Hey,</Text>
      <Text>
        <strong>{workspaceName}</strong> has used <strong>{used}</strong> of its{" "}
        <strong>{limit}</strong> free API requests this month. That counts successful key
        verifications and rate limit checks.
      </Text>
      <Text>Nothing is being rate limited yet. Requests are still going through.</Text>
      <Text>
        If this workspace stays on the free plan, we&apos;ll start rate limiting it in 3 business
        days. Upgrading to a paid API plan raises the limit and avoids that.
      </Text>

      <Section className="text-center py-3">
        <Button href={billingUrl} className="bg-gray-900 text-gray-50 rounded-lg px-7 py-3">
          Upgrade plan
        </Button>
      </Section>

      <Hr />
      <Text>
        Need help? Email <Link href="mailto:support@unkey.com">support@unkey.com</Link> or reply to
        this email.
      </Text>

      <Text className="text-xs">
        You are receiving this because you are an admin of this workspace.
      </Text>
      <Text className="text-xs">© {year} Unkey</Text>
    </Layout>
  );
}

FreeApiQuotaWarning.PreviewProps = {
  workspaceName: "Acme Inc",
  used: "150,000",
  limit: "150,000",
  billingUrl: "https://app.unkey.com/acme/settings/billing",
  year: "2026",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default FreeApiQuotaWarning;
