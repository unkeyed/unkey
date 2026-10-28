import { Layout } from "../src/components/layout";

export type Props = {
  date: string;
  source: string;
  url: string;
};

export function SecretScanningKeyDetected({ date, source, url }: Props) {
  return (
    <Layout
      title="Root key exposed on GitHub"
      preview="GitHub detected an exposed root key. Disable it in your dashboard."
      body={[
        "GitHub detected an exposed root key. Disable it in your dashboard, then create a replacement.",
      ]}
      action={{ label: "Open dashboard", href: "https://app.unkey.com" }}
      details={[
        { label: "Source", value: source },
        { label: "Detected", value: date },
      ]}
      sections={[
        {
          type: "links",
          links: [
            { label: "View on GitHub", href: url },
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
SecretScanningKeyDetected.PreviewProps = {
  date: "Thu Oct 22 2026",
  source: "content",
  url: "https://github.com/Flo4604/mono-repo-test/blob/e87412c7aa9a0958b129887f37ad3f023f289d22/key-leak-probe.txt",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default SecretScanningKeyDetected;
