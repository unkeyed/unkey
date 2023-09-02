import { Body } from "@react-email/body";
import { Container } from "@react-email/container";
import { Head } from "@react-email/head";
import { Heading } from "@react-email/heading";
import { Html } from "@react-email/html";
import { Link } from "@react-email/link";
import { Preview } from "@react-email/preview";
import { Tailwind } from "@react-email/tailwind";
import { Text } from "@react-email/text";

import React from "react";

export type Props = {
  name: string;
};

export function ResetWarning({ name }: Props) {
  return (
    <Tailwind>
      <Html className="font-sans text-gray-800">
        <Head />
        <Preview>Unkey: API Key Reset (Sunday)</Preview>
        <Body className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-center text-semibold">
              Unkey: API Key Reset (Sunday)
            </Heading>

            <Text>{name ? `Hi ${name},` : "Hi,"}</Text>
            <Text>
              We are writing to inform you that Unkey will be removing all API keys from
              applications on July 2nd 2023. While we know this is going to affect your applications
              we are doing this for the following reasons:
            </Text>
            <Text>1. Improving the Unkey API</Text>
            <Text>2. Improving the hashing algorithm that we used for enhanced security.</Text>
            <Text>
              We hope you understand as early adopters that we want to make Unkey the easiest and
              fastest way to allow users to interact with your API. This is a one time migration and
              won’t affect future adoption, if you have any questions just shoot us an email, or
              join the{" "}
              <Link
                target="_blank"
                className="text-blue-600 decoration-none"
                rel="noreferrer"
                href="https://discord.gg/qzTNdg3EVs"
              >
                Discord
              </Link>
              .
            </Text>
            <Text>Cheers,</Text>
            <Text>Andreas & James (Unkey Founders)</Text>
          </Container>
        </Body>
      </Html>
    </Tailwind>
  );
}

export default ResetWarning;
