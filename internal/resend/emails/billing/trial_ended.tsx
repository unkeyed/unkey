"use client";
import {
  Button,
  Container,
  Head,
  Heading,
  Hr,
  Html,
  Link,
  Section,
  Tailwind,
  Text,
} from "@react-email/components";
import React from "react";
export type Props = {
  username: string;
  workspaceName: string;
};

export function TrialEnded({ workspaceName = "workspaceName", username = "" }: Props) {
  return (
    <Tailwind>
      <Head />
      <Html className="font-sans text-zinc-800">
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
                When you upgrade to the Unkey Pro plan…it's simple. You get unlimited active keys
                and verifications:{" "}
              </Text>
              <Text>
                <li> 250 monthly active keys included (free users get 100 total)</li>
                <li> 10,000 verifications included (free users get 2,500 per month) </li>
                <li> Unlimited and free seats to invite your whole team</li>
                <li> Priority Support</li>
                <li> 90 days data retention</li>
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

export default TrialEnded;
