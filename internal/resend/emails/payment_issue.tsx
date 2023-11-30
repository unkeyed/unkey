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
export type Props = {
  username: string;
  date: string;
};

export function PaymentIssue({ username = "username", date = "date" }: Props) {
  return (
    <Tailwind>
      <Html className="font-sans text-zinc-800">
        <Head />
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">
              There was an issue with your payment.
            </Heading>
            <Text>Hey {username},</Text>
            <Text>
              We had trouble processing your payment on {date}. Please update your payment
              information below to prevent your account from being downgraded.
            </Text>

            <Container className="flex items-center justify-center my-8">
              <Button
                href="https://unkey.dev/app/settings/billing/stripe"
                className="px-4 py-2 text-white bg-black rounded"
              >
                Update payment information
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

export default PaymentIssue;
