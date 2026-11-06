const docsApiReferenceUrl = "https://www.unkey.com/docs/platform/api/overview";

export function composeAuthMd({
  hosted,
  apiUrl,
  dashboardUrl,
}: {
  hosted: string;
  apiUrl: string;
  dashboardUrl: string;
}): string {
  const apiOrigin = new URL(apiUrl).origin;
  const dashboardWorkspaceNewUrl = new URL("/new", dashboardUrl).toString();

  // Steps are named, not numbered, because WorkOS owns the hosted file's numbering.
  const section = [
    "## Use the token with Unkey",
    "",
    `Call the Unkey API at <${apiOrigin}>. Send the access token from the exchange step as \`Authorization: Bearer $ACCESS_TOKEN\`. API reference: <${docsApiReferenceUrl}>.`,
    "",
    "- Omit `resource` when you exchange the assertion. Unkey accepts only tokens issued for the default audience.",
    "- Unkey accepts tokens only from a claimed registration. Register with `service_auth` and complete the claim ceremony. Unkey rejects anonymous registrations.",
    "- The token acts as the user who claimed it. It carries that user's role in the workspace they selected on the claim page.",
    "- Root keys you create or reroll must expire no later than the access token. Set `expires` in Unix milliseconds, at or before the token's `exp`.",
    "",
    "When Unkey rejects a request:",
    "",
    "| Status | Message | What to do |",
    "| --- | --- | --- |",
    "| `401` | Invalid bearer token. | The access token expired or is malformed. Exchange the stored assertion for a new token. If the exchange fails, refresh the registration. |",
    "| `403` | This agent registration has not been claimed by a user. | Complete the claim ceremony with the user, then exchange the assertion again. |",
    `| \`403\` | Your organization does not have an active workspace. | The organization selected on the claim page has no Unkey workspace. Ask the user to create one at <${dashboardWorkspaceNewUrl}>, then retry for up to 30 seconds, because Unkey briefly caches the missing workspace. If it still fails, register again and ask the user to select the new workspace's organization on the claim page. |`,
    "| `403` | The user who claimed this agent is not a member of this workspace. | The user lost access to the workspace. Stop and tell the user. Retrying does not help. |",
    "| `400` | Expires is required and must not be later than the agent access token's expiration. | Set `expires` at or before the token's `exp`. The same limit applies when you reroll a root key. |",
    "| `503` | Unable to verify the bearer token right now. | Back off and retry the same request. |",
  ].join("\n");

  return `${hosted.trimEnd()}\n\n${section}\n`;
}
