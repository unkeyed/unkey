"use client";
import { Button } from "@react-email/button";
import { Heading } from "@react-email/heading";
import { Hr } from "@react-email/hr";
import { Link } from "@react-email/link";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
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
      <Heading className="font-sans text-3xl text-semibold text-center">Welcome to Unkey!</Heading>
      <Text>Hi there!</Text>
      <Text>
        I'm James, one of the co-founders of Unkey. We believe that Unkey's API management platform
        makes it easy to secure, manage and scale your API.
      </Text>
      <Section>
        <Text className="font-semibold">
          We know integrating a new system is overwhelming, so here are some resources to get you
          started:{" "}
        </Text>
        <ul className="pb-4">
          <li className="pt-4">
            {" "}
            <Link href="https://go.unkey.com/api-onboard">
              Unkey Public API Protection Quickstart Guide
            </Link>
          </li>
          <li className="pt-4">
            {" "}
            <Link href="https://go.unkey.com/ratelimit">Ratelimiting Quickstart Guide</Link>
          </li>
          <li className="pt-4">
            <Link href="https://unkey.com/docs/security"> Why is Unkey secure? </Link>
          </li>
          <li className="pt-4">
            {" "}
            <Link href="https://unkey.com/discord">Unkey Community Discord </Link>
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

export default WelcomeEmail;
