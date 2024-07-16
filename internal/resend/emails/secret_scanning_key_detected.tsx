
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
  username: string;
  date: string;
  url: string;
  type: string;
  source: string;
  key: string;
};

export function SecretScanningKeyDetected({ username, date, url, type, source, key }: Props) {
  return (
    <Tailwind config={tailwindConfig}>
      <Html className="font-sans text-zinc-800">
        <Head />
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">
              Alert! One of your keys was found to be leaked.
            </Heading>
            <Text>Hey {username},</Text>
            <Text>
              Github has found one of your keys has been leaked. Details are as follows:  
            </Text>
            <Hr />
              <Text>- Type: {type}</Text><Hr />
              <Text>- Source: {source}</Text><Hr />
              <Text>- Key: {key}</Text><Hr />
              <Text>- Date: {date}</Text><Hr />
              <Text>- URL: {url}</Text><Hr />
 
          

            <Container className="flex items-center justify-center my-8">
              <Button
                href={url}
                className="px-4 py-2 text-white bg-black rounded"
              >
                Leak Url
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
              Michael
            </Text>
          </Container>
        </Section>
      </Html>
    </Tailwind>
  );
}
SecretScanningKeyDetected.PreviewProps = {
  username: "MikePersonal",
  date: "7/12/2024",
  url: "http://unkey.com",
  type: "key",
  source: "commit",
  key: "test_12345",
} satisfies Props;
export default SecretScanningKeyDetected;
