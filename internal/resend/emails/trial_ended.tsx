"use client";
import { Button } from "@react-email/button";
import { Container } from "@react-email/container";
import { Head } from "@react-email/head";
import { Heading } from "@react-email/heading";
import { Hr } from "@react-email/hr";
import { Html } from "@react-email/html";
import { Link } from "@react-email/link";
import { Section } from "@react-email/section";
import { Tailwind } from "@react-email/tailwind";
import { Text } from "@react-email/text";
import React from "react";
import tailwindConfig from "../tailwind.config";
export type Props = {
  username: string;
  workspaceName: string;
};

export function TrialEnded({ workspaceName, username }: Props) {
  return (
    <Tailwind config={tailwindConfig}>
      <Html className="font-sans text-zinc-800">
        <Head />
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">
              Your workspace <strong>{workspaceName}</strong> has reached the end of its trial.
            </Heading>
            <Text>Hey {username},</Text>
            <Text>
              we hope you’ve enjoyed your two-week Unkey Pro trial for your workspace{" "}
              {workspaceName}. Your trial ended, add a payment method to keep all features of the
              pro plan.
            </Text>

            <Section>
              <Text className="font-semibold">
                When you upgrade to the Unkey Pro plan…it's simple:{" "}
              </Text>
              <Text>
                <li> 1M monthly active keys included (free users get 1k total)</li>
                <li> 150k monthly verifications included (free users get 2.5k per month) </li>
                <li> 2.5M monthly ratelimits included (free users get 100k per month) </li>
                <li> Unlimited and free seats to invite your whole team</li>
                <li> 90-day analytics retention</li>
                <li> 90-day audit log retention</li>
                <li> Priority Support</li>
              </Text>
            </Section>

            <Container className="flex items-center justify-center my-8">
              <Button
                href="https://unkey.dev/app/settings/billing"
                className="px-4 py-2 text-white bg-black rounded"
              >
                Upgrade Now
              </Button>
            </Container>

            <Hr />

            <Text>
              Need help? Please reach out to{" "}
              <Link href="mailto:support@unkey.dev">support@unkey.dev</Link> or just reply to this
              email.
            </Text>

            <Text>
              Cheers,
              <br />
              Andreas
            </Text>
          </Container>
        </Section>
      </Html>
    </Tailwind>
  );
}

TrialEnded.PreviewProps = {
  username: "Spongebob Squarepants",
  workspaceName: "Krusty crab",
} satisfies Props;

export default TrialEnded;
