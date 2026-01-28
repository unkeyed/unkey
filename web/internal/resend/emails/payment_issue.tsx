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
  date: string;
};

export function PaymentIssue({ username, date }: Props) {
  return (
    <Layout>
      <Heading className="font-sans text-3xl text-semibold text-center">
        There was an issue with your payment.
      </Heading>
      <Text>Hey {username},</Text>
      <Text>
        We had trouble processing your payment on {date}. Please update your payment information
        below to prevent your account from being downgraded.
      </Text>

      <Section className="text-center py-3">
        <Button
          href="https://app.unkey.com/settings/billing/stripe"
          className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3"
        >
          Update payment information
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

PaymentIssue.PreviewProps = {
  username: "Mr. Pilkington",
  date: "Tue Oct 01 2024", // Date().toDateString
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default PaymentIssue;
