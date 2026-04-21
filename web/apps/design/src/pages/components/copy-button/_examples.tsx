import { Code, CopyButton } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function StandaloneExample() {
  return (
    <Preview>
      <CopyButton value="unkey_3ZaM7pQ1xK9VdLb2wTgRcY" />
    </Preview>
  );
}

export function InsideCodeExample() {
  return (
    <Preview>
      <Code
        copyButton={<CopyButton value="unkey_3ZaM7pQ1xK9VdLb2wTgRcY" />}
      >
        <span>unkey_3ZaM7pQ1xK9VdLb2wTgRcY</span>
      </Code>
    </Preview>
  );
}

export function WithToastMessageExample() {
  return (
    <Preview>
      <CopyButton
        value="whsec_9f8a1c2b3d4e5f6a7b8c9d0e1f2a3b4c"
        toastMessage="Signing secret copied. Treat it like a password."
      />
    </Preview>
  );
}
