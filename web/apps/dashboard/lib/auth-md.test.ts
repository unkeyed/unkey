import { describe, expect, it } from "vitest";
import { composeAuthMd } from "./auth-md";

const hosted = [
  "# auth.md",
  "",
  "## 4. Exchange the assertion for an access token",
  "",
  "Send it as `Authorization: Bearer $ACCESS_TOKEN`.",
  "",
].join("\n");

const urls = { apiUrl: "https://api.unkey.com", dashboardUrl: "https://app.unkey.com" };

describe("composeAuthMd", () => {
  it("keeps the hosted file verbatim and appends the Unkey section after it", () => {
    const composed = composeAuthMd({ hosted, ...urls });

    expect(composed.startsWith(`${hosted.trimEnd()}\n\n## Use the token with Unkey\n`)).toBe(true);
    expect(composed.endsWith("| Back off and retry the same request. |\n")).toBe(true);
  });

  it("tells the agent where to send the token and the rules Unkey enforces", () => {
    const composed = composeAuthMd({ hosted, ...urls });

    expect(composed).toContain(
      "Call the Unkey API at <https://api.unkey.com>. Send the access token from the exchange step as `Authorization: Bearer $ACCESS_TOKEN`.",
    );
    expect(composed).toContain(
      "API reference: <https://www.unkey.com/docs/platform/api/overview>.",
    );
    expect(composed).toContain("Omit `resource` when you exchange the assertion.");
    expect(composed).toContain("Register with `service_auth` and complete the claim ceremony.");
    expect(composed).toContain("Unkey rejects anonymous registrations.");
    expect(composed).toContain(
      "Root keys you create or reroll must expire no later than the access token.",
    );
  });

  it("maps each error Unkey returns to agents onto a recovery", () => {
    const composed = composeAuthMd({ hosted, ...urls });

    for (const row of [
      "| `401` | Invalid bearer token. |",
      "| `403` | This agent registration has not been claimed by a user. |",
      "| `403` | Your organization does not have an active workspace. |",
      "| `403` | The user who claimed this agent is not a member of this workspace. |",
      "| `400` | Expires is required and must not be later than the agent access token's expiration. |",
      "| `503` | Unable to verify the bearer token right now. |",
    ]) {
      expect(composed).toContain(row);
    }
    expect(composed).toContain(
      "Ask the user to create one at <https://app.unkey.com/new>, then retry for up to 30 seconds",
    );
  });

  it("refers to hosted steps by name so WorkOS can renumber them", () => {
    const composed = composeAuthMd({ hosted, ...urls });
    const section = composed.slice(hosted.trimEnd().length);

    expect(section).not.toMatch(/step \d/i);
  });

  it("normalizes trailing slashes on configured URLs", () => {
    const composed = composeAuthMd({
      hosted,
      apiUrl: "http://localhost:7070/",
      dashboardUrl: "http://localhost:3000/",
    });

    expect(composed).toContain("Call the Unkey API at <http://localhost:7070>.");
    expect(composed).toContain("<http://localhost:3000/new>");
  });
});
