import { Body } from "@react-email/body";
import { Container } from "@react-email/container";
import { Head } from "@react-email/head";
import { Html } from "@react-email/html";
import { Link } from "@react-email/link";
import { Section } from "@react-email/section";
import { Tailwind } from "@react-email/tailwind";
import { Text } from "@react-email/text";
// biome-ignore lint/style/useImportType: need access to `children`, not just the type
import React from "react";
import type { ReactNode } from "react";

interface LayoutProps {
  children: ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => (
  <Html>
    <Tailwind>
      <Head />
      <Body className="bg-white font-sans text-zinc-800">
        <Container className="container mx-auto p-6">
          <Section className="mx-auto p-6 bg-gray-50">{children}</Section>
          <Section className="container mx-auto p-6 text-center font-semibold">
            <Text>
              Connect with us on social media!
              <br />
              <Link href="https://x.com/unkeydev">X (formerly Twitter)</Link> |{" "}
              <Link href="https://unkey.com/discord">Discord</Link> |{" "}
              <Link href="https://unkey.com/github">GitHub</Link>
            </Text>
          </Section>
        </Container>
      </Body>
    </Tailwind>
  </Html>
);
