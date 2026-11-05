import { Fragment } from "react";
import { Emphasis } from "../src/components/emphasis";
import { Layout } from "../src/components/layout";

export type Props = {
  date: string;
  source: string;
  url: string;
};

export function TruffleHogKeyDetected({ date, source, url }: Props) {
  return (
    <Layout
      title="Exposed Unkey root key revoked"
      preview="TruffleHog found an exposed Unkey root key. We revoked it, so requests using it now fail."
      body={[
        <Fragment key="revocation">
          TruffleHog found an Unkey root key exposed publicly at {source}, and we revoked it on{" "}
          <Emphasis>{date}</Emphasis>, so requests using it now fail.
        </Fragment>,
        "Create a replacement root key and update your applications to use it.",
      ]}
      action={{ label: "Open dashboard", href: "https://app.unkey.com" }}
      details={[
        { label: "Source", value: source },
        { label: "Revoked", value: date },
      ]}
      sections={[
        {
          type: "links",
          links: [
            { label: "View where it was found", href: url },
            {
              label: "Root key recovery guide",
              href: "https://www.unkey.com/docs/platform/root-keys/overview#if-a-root-key-leaks",
            },
          ],
        },
      ]}
    />
  );
}
TruffleHogKeyDetected.PreviewProps = {
  date: "Thu Nov 05 2026",
  source: "Docker",
  url: "https://hub.docker.com/r/acme/api",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default TruffleHogKeyDetected;
