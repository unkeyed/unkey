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
  workspaceName: string;
  deprecatedEndpoints: string[];
};

export function ApiV1Migration({
  username = "User",
  workspaceName = "Your Workspace",
  deprecatedEndpoints = [],
}: Props) {
  return (
    <Layout>
      <Heading className="font-sans text-3xl text-semibold text-center">
        Action Required: Migrate from API v1 to v2
      </Heading>
      <Text>Hey {username},</Text>
      <Text>
        This is a follow-up to our previous email in August about the deprecation of our API v1.
        We're reaching out because your workspace <strong>{workspaceName}</strong> is still using
        deprecated v1 endpoints.
      </Text>

      <Text className="font-semibold text-red-600">
        Important: <span className="font-mono">api.unkey.dev/v1</span> will be discontinued on
        January 1st, 2026.
      </Text>

      <Section>
        <Text className="font-semibold">
          Your workspace is currently using the following deprecated v1 endpoints:
        </Text>
        <ul className="pb-4">
          {deprecatedEndpoints.map((endpoint) => (
            <li key={endpoint} className="pb-2">
              <code className="bg-gray-100 px-2 py-1 rounded text-sm">{endpoint}</code>
            </li>
          ))}
        </ul>
      </Section>

      <Section className="text-center py-3">
        <Button
          href="https://www.unkey.com/docs/api-reference/v1/migration"
          className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3"
        >
          View Migration Guide
        </Button>
      </Section>

      <Text>
        Our migration guide includes step-by-step instructions, code examples, and a complete
        mapping of v1 to v2 endpoints.
      </Text>

      <Hr />

      <Text>
        Need help with your migration? Please reach out to{" "}
        <Link href="mailto:support@unkey.com">support@unkey.com</Link> or just reply to this email.
        Our team is ready to help with anything.
      </Text>

      <Signature signedBy="Andreas" />
    </Layout>
  );
}

ApiV1Migration.PreviewProps = {
  username: "John Doe",
  workspaceName: "Acme Corp",
  deprecatedEndpoints: ["/v1/keys.create", "/v1/keys.verify", "/v1/apis.create"],
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default ApiV1Migration;
