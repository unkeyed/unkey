import { Heading } from "@react-email/heading";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
// biome-ignore lint/correctness/noUnusedImports: react-email needs this imported
import React from "react";
import { Layout } from "../src/components/layout";
import { Signature } from "../src/components/signature";

export type Props = {
  otp: string;
  type: "sign-in" | "email-verification" | "forget-password";
};

function getPreviewText(type: Props["type"]) {
  switch (type) {
    case "sign-in":
      return "Use this code to sign in to your Unkey account";
    case "email-verification":
      return "Use this code to verify your email address";
    case "forget-password":
      return "Use this code to reset your Unkey password";
  }
}

export function OtpVerificationEmail({ otp = "123456", type = "sign-in" }: Props) {
  const previewText = getPreviewText(type);

  return (
    <Layout>
      <Heading className="font-sans text-3xl text-semibold text-center">
        Your verification code
      </Heading>

      <Text>{previewText}. This code expires in 10 minutes.</Text>

      <Section className="text-center py-6">
        <Text className="text-4xl font-mono font-bold tracking-widest m-0 bg-gray-100 rounded-lg py-4">
          {otp}
        </Text>
      </Section>

      <Text className="text-gray-500 text-sm">
        If you didn't request this code, you can safely ignore this email.
      </Text>

      <Signature signedBy="The Unkey Team" />
    </Layout>
  );
}

OtpVerificationEmail.PreviewProps = {
  otp: "847293",
  type: "sign-in",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default OtpVerificationEmail;
