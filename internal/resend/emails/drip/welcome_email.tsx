"use client";
import {
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

export function WelcomeEmail() {
  return (
    <Tailwind>
      <Head />
      <Html className="font-sans text-zinc-800">
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">Welcome to Unkey!</Heading>
            <Text>Hi there!</Text>
            <Text>
              My name is James. I am one of the co-founders of Unkey. We believe that Unkey's API
              management platform makes it easy to secure, manage and scale your API.
            </Text>
            <Section>
              <Text className="font-semibold">
                We know integrating a new system is overwhelming, so here are some resources to get
                you started:{" "}
              </Text>
              <Text>
                <li key="quickstart">
                  {" "}
                  <Link href="https://unkey.dev/docs/onboarding">Unkey Quickstart Guide</Link>
                </li>
                <li key="security">
                  <Link href="https://unkey.dev/docs/security"> Why is Unkey secure? </Link>
                </li>
                <li key="discord">
                  {" "}
                  <Link href="https://unkey.dev/discord">Unkey Community Discord </Link>
                </li>
              </Text>
            </Section>
            <Hr />
            <Text>
              We love feedback, so feel free to respond to this email as you start using Unkey. We
              read and reply to every single one.
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

export default WelcomeEmail;
