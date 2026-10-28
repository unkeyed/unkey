import { Layout } from "../src/components/layout";

export function WelcomeEmail() {
  return (
    <Layout
      title="Welcome to Unkey"
      preview="Deploy your first API with built-in auth, rate limiting, and analytics."
      body={[
        "I'm James, one of Unkey's co-founders. Unkey helps you ship a live, multi-region API with auth, rate limiting, and analytics built in.",
        "Connect a GitHub repo or run unkey deploy to get started.",
      ]}
      action={{ label: "Deploy your first API", href: "https://app.unkey.com/" }}
      sections={[
        {
          type: "links",
          links: [
            {
              label: "Deploy your first app",
              href: "https://www.unkey.com/docs/quickstart/deploy",
            },
            {
              label: "How deployments work",
              href: "https://www.unkey.com/docs/build-and-deploy/overview",
            },
          ],
        },
        {
          type: "text",
          text: "Looking for API management? Issue, verify, and revoke API keys from any backend, with no deployment required.",
        },
        {
          type: "links",
          links: [
            {
              label: "Get started with API management",
              href: "https://www.unkey.com/docs/platform/apis/introduction",
            },
          ],
        },
        {
          type: "text",
          text: "How did you hear about Unkey? Reply to this email with questions or feedback. I read and reply to every one.",
        },
      ]}
      signedBy="James"
      socialLinks={[
        { label: "X", href: "https://x.com/unkeydev" },
        { label: "Discord", href: "https://unkey.com/discord" },
        { label: "GitHub", href: "https://unkey.com/github" },
      ]}
    />
  );
}

WelcomeEmail.PreviewProps = {};

// biome-ignore lint/style/noDefaultExport: the email dev preview and render test load default exports
export default WelcomeEmail;
