import { Fragment } from "react";
import { Emphasis } from "../src/components/emphasis";
import { Layout } from "../src/components/layout";

export type Props = {
  date: string;
  source: string;
  url: string;
  keyId: string;
};

export function TruffleHogApiKeyDetected({ date, source, url, keyId }: Props) {
  return (
    <Layout
      title="Exposed API key disabled"
      preview="TruffleHog found one of the API keys your workspace issued exposed publicly. It is now disabled, so requests using it fail."
      body={[
        <Fragment key="disabled">
          TruffleHog found one of the API keys your workspace issued exposed publicly at {source} on{" "}
          <Emphasis>{date}</Emphasis>. The key is now disabled, so requests using it fail.
        </Fragment>,
        <Fragment key="replacement">
          Find the user it belongs to by key ID <Emphasis>{keyId}</Emphasis> and issue them a
          replacement. Then delete the disabled key. Do not enable it again.
        </Fragment>,
      ]}
      action={{ label: "Open dashboard", href: "https://app.unkey.com" }}
      details={[
        { label: "Source", value: source },
        { label: "Key ID", value: keyId },
        { label: "Reported", value: date },
      ]}
      sections={[
        {
          type: "links",
          links: [
            { label: "View where it was found", href: url },
            {
              label: "Handling a leaked user key",
              href: "https://www.unkey.com/docs/api-management/security/overview#leaked-credentials",
            },
          ],
        },
      ]}
    />
  );
}
TruffleHogApiKeyDetected.PreviewProps = {
  date: "Thu Nov 05 2026",
  source: "Docker",
  url: "https://hub.docker.com/r/acme/api",
  keyId: "key_2cGKbMxRyIzhCxo1Idjz8q",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default TruffleHogApiKeyDetected;
