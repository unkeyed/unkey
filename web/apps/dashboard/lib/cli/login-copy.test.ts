import { describe, expect, it } from "vitest";
import { loginCopy } from "./login-copy";

describe("loginCopy", () => {
  it("asks a CLI login to compare the code from the terminal", () => {
    const copy = loginCopy({ oauth: undefined });

    expect(copy.confirmCode).toBe(true);
    expect(copy.title).toBe("Sign in to the Unkey CLI");
    expect(copy.authorizeLabel).toBe("Authorize CLI");
  });

  it("names an OAuth client next to the host that vouches for it and the server it will use", () => {
    const copy = loginCopy({
      oauth: {
        clientName: "Claude Code",
        clientHost: "claude.ai",
        resource: "https://mcp.unkey.com/compute",
      },
    });

    expect(copy.confirmCode).toBe(false);
    expect(copy.title).toBe("Authorize Claude Code");
    expect(copy.description).toBe(
      "Claude Code (claude.ai) wants to use Unkey through mcp.unkey.com/compute.",
    );
    expect(copy.noPermissions).toBe("Your role has no permissions it can give to Claude Code.");
    expect(copy.approvedDescription).toBe("Returning you to Claude Code. You can close this tab.");
  });
});
