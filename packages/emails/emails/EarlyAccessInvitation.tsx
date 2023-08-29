import { Body } from "@react-email/body";
import { Button } from "@react-email/button";
import { Container } from "@react-email/container";
import { Head } from "@react-email/head";
import { Heading } from "@react-email/heading";
import { Html } from "@react-email/html";
import { Link } from "@react-email/link";
import { Preview } from "@react-email/preview";
import { Section } from "@react-email/section";
import { Tailwind } from "@react-email/tailwind";
import { Text } from "@react-email/text";
import React from "react";

export type Props = {
  inviteLink: string;
};

export function EarlyAccessInvitation({ inviteLink }: Props) {
  return (
    <Tailwind>
      <Html className="font-sans text-gray-800">
        <Head />
        <Preview>Join the Early Access on unkey.dev</Preview>
        <Body className="bg-white">
          <Container className="container mx-auto">
            {/* <Section className="mt-8">
              <Img
                src="https://planetfall.io/logo.png"
                width="32"
                height="32"
                alt="Planetfall's Logo"
              />
            </Section> */}
            <Heading className="font-sans text-2xl text-center text-semibold">
              Join the <strong>Early Access</strong> on <strong>unkey.dev</strong>
            </Heading>

            <Section
              style={{
                textAlign: "center",
                marginTop: "26px",
                marginBottom: "26px",
              }}
            >
              <Button
                className="px-8 py-4 font-medium text-white rounded bg-gray-900"
                href={inviteLink}
              >
                Sign Up
              </Button>
            </Section>
            <Text>
              or copy and paste this URL into your browser:{" "}
              <Link
                href={inviteLink}
                target="_blank"
                className="text-blue-600 decoration-none"
                rel="noreferrer"
              >
                {inviteLink}
              </Link>
            </Text>

            <Text>
              Unkey is the open source API Key management solution. Allowing you to create, manage
              and validate API Keys for your users and comes withwith security and speed in mind.
            </Text>

            <Text>We also just launched a discord server, come say hi:</Text>
            <Link
              target="_blank"
              className="text-blue-600 decoration-none"
              rel="noreferrer"
              href="https://discord.gg/qzTNdg3EVs"
            >
              https://discord.gg/qzTNdg3EVs
            </Link>
          </Container>
        </Body>
      </Html>
    </Tailwind>
  );
}

export default EarlyAccessInvitation;
