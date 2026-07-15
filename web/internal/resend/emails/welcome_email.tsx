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

export function WelcomeEmail() {
  return (
    <Layout>
      <Heading className="font-sans text-3xl font-semibold text-center">Welcome to Unkey!</Heading>
      <Text>Hi there!</Text>
      <Text>
        I'm James, one of Unkey's co-founders. Unkey is the fastest way to get an API into
        production: push your code, get a live, multi-region API with auth, rate limiting, and
        analytics built in. No infrastructure to stitch together.
      </Text>
      <Section>
        <Text className="font-semibold">The fastest way to see it in action:</Text>
        <ul className="pb-4 text-sm">
          <li className="pt-4">
            {" "}
            <Link href="https://www.unkey.com/docs/quickstart/deploy">Deploy your first app</Link> -
            Connect a GitHub repo or run <code>unkey deploy</code> and have a live URL in minutes,
            with preview environments for every branch and instant rollbacks.
          </li>
          <li className="pt-4">
            {" "}
            <Link href="https://www.unkey.com/docs/build-and-deploy/overview">
              How deployments work
            </Link>{" "}
            - Multi-region routing, immutable versions, automatic domains, and built-in protection
            in front of every request.
          </li>
          <li className="pt-4">
            {" "}
            <Link href="https://unkey.com/discord">Unkey Community Discord </Link> - Connect with
            other developers shipping APIs on Unkey, ask questions, and talk to the team directly.
          </li>
        </ul>
      </Section>
      <Section className="text-center py-3">
        <Button
          href="https://app.unkey.com/"
          className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3"
        >
          Deploy your first API
        </Button>
      </Section>
      <Hr />
      <Text className="text-sm">
        Looking for API management?{" "}
        <Link href="https://www.unkey.com/docs/platform/apis/introduction">Start here</Link> -
        issue, verify, and revoke API keys from any backend, no deployment required.
      </Text>
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
