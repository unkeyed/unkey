import { Layout } from "../src/components/layout";

export type Props = {
  username: string;
  workspaceName: string;
  deprecatedEndpoints: string[];
};

export function ApiV1Migration({
  username = "User",
  workspaceName = "Your Workspace",
  deprecatedEndpoints = [],
}: Props) {
  return (
    <Layout
      title="Migrate from API v1 to v2"
      preview={`${workspaceName} still uses API v1. Migrate before January 1st, 2026.`}
      body={[
        `${username}, your workspace ${workspaceName} is still using deprecated v1 endpoints. This follows our August email about the API v1 deprecation.`,
        "api.unkey.dev/v1 will be discontinued on January 1st, 2026. Migrate to API v2 before then.",
      ]}
      action={{
        label: "View migration guide",
        href: "https://www.unkey.com/docs/platform/apis/migrations/introduction",
      }}
      sections={[
        { type: "list", title: "Deprecated endpoints in use", items: deprecatedEndpoints },
        {
          type: "text",
          text: "The migration guide includes step-by-step instructions, code examples, and a complete mapping of v1 to v2 endpoints.",
        },
      ]}
    />
  );
}

ApiV1Migration.PreviewProps = {
  username: "John Doe",
  workspaceName: "Acme Corp",
  deprecatedEndpoints: ["/v1/keys.create", "/v1/keys.verify", "/v1/apis.create"],
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default ApiV1Migration;
