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
  date: string;
  source: string;
  type: string;
  url: string;
};

export function SecretScanningKeyDetected({ date, source, type, url }: Props) {
  return (
    <Tailwind config={tailwindConfig}>
      <Html className="font-sans text-zinc-800">
        <Head />
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">
              Alert! One of your keys was found to be leaked.
            </Heading>
            <Text>Hello</Text>
            <Text>Github has found one of your keys has been leaked. Details are as follows:</Text>
            <Hr />
            <Text>- Type: {type}</Text>
            <Hr />
            <Text>- Source: {source}</Text>
            <Hr />
            <Text>- Date: {date}</Text>
            <Hr />
            <Text>- URL: {url}</Text>
            <Hr />

            <Container className="flex items-center justify-center my-8">
              <Button href={url} className="px-4 py-2 text-white bg-black rounded">
                Leak Url
              </Button>
            </Container>
            <Hr />
            <Text>
              Please reach out to us to discuss your options. If we do not hear back from you the
              key will be deleted. <Link href="mailto:support@unkey.dev">support@unkey.dev</Link> or
              just reply to this email.
            </Text>

            <Text>
              Cheers,
              <br />
              James
            </Text>
          </Container>
        </Section>
      </Html>
    </Tailwind>
  );
}
SecretScanningKeyDetected.PreviewProps = {
  date: "7/12/2024",
  type: "key",
  source: "commit",
  url: "http://unkey.com",
} satisfies Props;

export default SecretScanningKeyDetected;
