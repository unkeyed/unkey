import { Button } from "@react-email/button";
import { Heading } from "@react-email/heading";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
// biome-ignore lint/correctness/noUnusedImports: react-email needs this imported
import React from "react";
import { Layout } from "../src/components/layout";
import { Signature } from "../src/components/signature";

export type Props = {
  inviterEmail: string;
  organizationName: string;
  invitationUrl: string;
};

export function OrgInvitationEmail({
  inviterEmail = "inviter@example.com",
  organizationName = "Acme Corp",
  invitationUrl = "https://app.unkey.com/auth/accept-invitation?token=abc123",
}: Props) {
  return (
    <Layout>
      <Heading className="font-sans text-3xl text-semibold text-center">
        You're invited to join {organizationName}
      </Heading>

      <Text>
        {inviterEmail} has invited you to join the <strong>{organizationName}</strong> workspace on
        Unkey.
      </Text>

      <Section className="text-center py-3">
        <Button href={invitationUrl} className="bg-gray-900 text-gray-50 rounded-lg p-3 w-2/3">
          Accept Invitation
        </Button>
      </Section>

      <Text className="text-gray-500 text-sm">
        If you don't want to join this workspace, you can safely ignore this email.
      </Text>

      <Signature signedBy="The Unkey Team" />
    </Layout>
  );
}

OrgInvitationEmail.PreviewProps = {
  inviterEmail: "james@unkey.com",
  organizationName: "Acme Corp",
  invitationUrl: "https://app.unkey.com/auth/accept-invitation?token=abc123",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default OrgInvitationEmail;
