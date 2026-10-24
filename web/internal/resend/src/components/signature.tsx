import { Text } from "@react-email/text";
// biome-ignore lint/style/useImportType: not just the type
import React from "react";
import { styles } from "./theme";

interface SignatureProps {
  signedBy: string;
}

export const Signature: React.FC<SignatureProps> = ({ signedBy }) => (
  <Text className="email-copy" style={styles.text}>
    Cheers,
    <br />
    {signedBy}
  </Text>
);
