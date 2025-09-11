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
  date: string;
};

export function WelcomeEmail() {
  return (
    <Layout>
      <Heading className="font-sans text-3xl font-semibold text-center">Welcome to Unkey!</Heading>
      <Text>Hi there!</Text>
      <Text>
        I'm James, one of Unkey's co-founders. Unkey's API Development platform is the fastest way
        from idea to production.
      </Text>
      <Section>
        <Text className="font-semibold">
          To support your journey, we’ve compiled a list of essential resources:
        </Text>
        <ul className="pb-4 text-sm">
          <li className="pt-4">
            {" "}
            <Link href="https://go.unkey.com/api-onboard">Quickstart Guides</Link> - Our complete
            series of guides will help you integrate Unkey step by step.
          </li>
          <li className="pt-4">
            <Link href="https://www.unkey.com/docs/api-reference/v2/overview">
              {" "}
              API Documentation
            </Link>{" "}
            - Our API reference documentation will help you understand and use our API features to
            their fullest potential.
          </li>
          <li className="pt-4">
            {" "}
            <Link href="https://unkey.com/discord">Unkey Community Discord </Link> - Connect with
            other users, share insights, ask questions, and find solutions within our community.
          </li>
        </ul>
      </Section>

      <Section className="text-center py-3">
        <Button
          href="https://app.unkey.com/"
          className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3"
        >
          Go to your dashboard
        </Button>
      </Section>

      <Hr />
      <Text>Also, just curious - how did you hear about Unkey?</Text>

      <Signature signedBy="James" />
      <Text className="text-xs">
        P.S. - if you have any questions or feedback, reply to this email. I read and reply to every
        single one.
      </Text>
    </Layout>
  );
}
