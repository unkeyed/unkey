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
  date: string;
  source: string;
  url: string;
};

export function SecretScanningKeyDetected({ date, source, url }: Props) {
  return (
    <Layout>
      <Heading className="font-sans text-3xl text-semibold text-center">
        Warning! One of your keys was leaked!
      </Heading>
      <Text>Hi there!</Text>
      <Text>Github found that one of your keys has been leaked. Details are as follows:</Text>
      <ul className="pb-4">
        <li className="pt-4">
          {" "}
          <strong>Source:</strong> {source}{" "}
        </li>
        <li className="pt-4">
          {" "}
          <strong>Date:</strong> {date}{" "}
        </li>
        <li className="pt-4">
          {" "}
          <strong>URL:</strong> {url}
        </li>
      </ul>
      <Section className="text-center py-3">
        <Button href={url} className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3">
          Go to source
        </Button>
      </Section>
      <Hr />
      <Text>
        You can disable the Root Key in your dashboard by following our docs available at{" "}
        <Link href="https://www.unkey.com/docs/security/root-keys">
          https://www.unkey.com/docs/security/root-keys
        </Link>
        .
      </Text>
      <Text>
        Need help? Please reach out to{" "}
        <Link href="mailto:support@unkey.com">support@unkey.com</Link> or just reply to this email.
      </Text>
      <Signature signedBy="James" />
    </Layout>
  );
}
SecretScanningKeyDetected.PreviewProps = {
  date: "Tue Oct 01 2024", // Date().toDateString
  source: "commit",
  url: "http://unkey.com",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default SecretScanningKeyDetected;
