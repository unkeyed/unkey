"use client";
import {
  Button,
  Container,
  Head,
  Heading,
  Hr,
  Html,
  Section,
  Tailwind,
  Text,
} from "@react-email/components";
import React from "react";
export type Props = {
  username: string;
};

export function SubscriptionEnded({ username = "username" }: Props) {
  return (
    <Tailwind>
      <Head />
      <Html className="font-sans text-zinc-800">
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">
              Your Unkey subscription has ended.
            </Heading>
            <Text>Hey {username},</Text>
            <Text>
              We're reaching out to let you know that your trial period has come to an end for Unkey
              Pro. We have downgraded the workspace to free, which means you lose access to the
              workspace, but we will retain all your team and key data.
            </Text>
            <Text>
              If you want to continue with Unkey Pro, click the button below, and you can add your
              credit card.
            </Text>

            <Container className="flex items-center justify-center my-8">
              <Button
                href="https://unkey.dev/app/settings/billing/stripe"
                className="px-4 py-2 text-white bg-black rounded"
              >
                Upgrade Now
              </Button>
            </Container>

            <Hr />
            <Text>
              If you have any feedback, please reply to this email. We would love to hear all about
              it.
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

export default SubscriptionEnded;
