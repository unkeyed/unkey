"use client";
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
export type Props = {
  username: string;
  date: string;
};

export function WelcomeEmail() {
  return (
    <Tailwind>
      <Html className="font-sans text-zinc-800">
        <Head />
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
                <li>
                  {" "}
                  <Link href="https://unkey.dev/docs/onboarding">Unkey Quickstart Guide</Link>
                </li>
                <li>
                  <Link href="https://unkey.dev/docs/security"> Why is Unkey secure? </Link>
                </li>
                <li>
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
