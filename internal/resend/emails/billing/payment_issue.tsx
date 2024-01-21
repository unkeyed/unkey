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
  date: string;
};

export function PaymentIssue({ username = "username", date = "date" }: Props) {
  return (
    <Tailwind>
      <Head />
      <Html className="font-sans text-zinc-800">
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
