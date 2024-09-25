"use client";
import { Button } from "@react-email/button";
import { Heading } from "@react-email/heading";
import { Hr } from "@react-email/hr";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
import React from "react";
import { Layout } from "../src/components/layout";
import { Signature } from "../src/components/signature";
export type Props = {
  username: string;
};

export function SubscriptionEnded({ username }: Props) {
  return (
    <Layout>
      <Heading className="font-sans text-3xl text-semibold text-center">
        Your Unkey subscription has ended.
      </Heading>
      <Text>Hey {username},</Text>
      <Text>
        We're reaching out to let you know that your trial period has come to an end for Unkey Pro.
        We have downgraded the workspace to Free, which means you lose access to the workspace, but
        we will retain all your team and key data.
      </Text>
      <Text>
        If you want to continue with Unkey Pro, click the button below, and you can add your credit
        card.
      </Text>

      <Section className="text-center py-3">
        <Button
          href="https://unkey.com/app/settings/billing"
          className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3"
        >
          Upgrade Now
        </Button>
      </Section>

      <Hr />
      <Text>
        If you have any feedback, please reply to this email. We would love to hear all about it.
      </Text>

      <Signature signedBy="Andreas" />
    </Layout>
  );
}

SubscriptionEnded.PreviewProps = {
  username: "Mike Wazowski",
} satisfies Props;

export default SubscriptionEnded;
