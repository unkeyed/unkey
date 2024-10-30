import { Text } from "@react-email/text";
// biome-ignore lint/style/useImportType: not just the type
import React from "react";

interface SignatureProps {
  signedBy: string;
}

export const Signature: React.FC<SignatureProps> = ({ signedBy }) => (
  <Text className="font-semibold">
    Cheers,
    <br />
    {signedBy}
  </Text>
);
