"use client";
import { Button } from "@react-email/button";
import { Heading } from "@react-email/heading";
import { Hr } from "@react-email/hr";
import { Link } from "@react-email/link";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
// biome-ignore lint/correctness/noUnusedImports: react-email needs this imported
import React from "react";
import { Layout } from "../src/components/layout";
import { Signature } from "../src/components/signature";

export type Props = {
  username: string;
  workspaceName: string;
};

export function TrialEnded({ workspaceName, username }: Props) {
  return (
    <Layout>
      <Heading className="font-sans text-3xl text-semibold text-center">
        Your workspace <strong>{workspaceName}</strong> has reached the end of its trial.
      </Heading>
      <Text>Hey {username},</Text>
      <Text>
        We hope you've enjoyed your two-week Unkey Pro trial for your workspace{" "}
        <strong>{workspaceName}</strong>.
      </Text>

      <Text>
        Since your trial ended, please add a payment method to keep all features of the Pro plan.
      </Text>

      <Section>
        <Text className="font-semibold">
          It's simple to upgrade and enjoy the benefits of the Unkey Pro plan:
        </Text>
        <ul>
          <li className="pb-4">
            {" "}
            1M monthly active keys included{" "}
            <span className="italic text-sm">(free users only get 1k total)</span>
          </li>
          <li className="pb-4">
            {" "}
            150k monthly verifications included{" "}
            <span className="italic text-sm">(free users only get 2.5k per month)</span>
          </li>
          <li className="pb-4">
            {" "}
            2.5M monthly ratelimits included{" "}
            <span className="italic text-sm">(free users only get 100k per month)</span>
          </li>
        </ul>
        <Text className="font-semibold">Pro workspaces also receive:</Text>
        <ul>
          <li className="pb-4">
            {" "}
            Unlimited seats at no additional cost so you can invite your whole team
          </li>
          <li className="pb-4"> 90-day analytics retention</li>
          <li className="pb-4"> 90-day audit log retention</li>
          <li className="pb-4"> Priority Support</li>
        </ul>
      </Section>

      <Section className="text-center py-3">
        <Button
          href="https://app.unkey.com/settings/billing"
          className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3"
        >
          Upgrade now
        </Button>
      </Section>

      <Hr />

      <Text>
        Need help? Please reach out to{" "}
        <Link href="mailto:support@unkey.dev">support@unkey.dev</Link> or just reply to this email.
      </Text>

      <Signature signedBy="James" />
    </Layout>
  );
}

TrialEnded.PreviewProps = {
  username: "Spongebob Squarepants",
  workspaceName: "Krusty crab",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default TrialEnded;
