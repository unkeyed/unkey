import { describe, expect, it } from "vitest";
import { sourceChoices } from "./source-choice-state";

describe("sourceChoices", () => {
  it("offers both sources to a new app", () => {
    expect(sourceChoices({ lockedTo: null, needsGithub: false })).toEqual([
      {
        kind: "git",
        action: "pick",
        title: "GitHub repository",
        description: "Build from source and deploy each push",
      },
      {
        kind: "oci",
        action: "pick",
        title: "Container image",
        description: "Run a prebuilt image from a public registry",
      },
    ]);
  });

  it("asks to connect GitHub before importing a repository", () => {
    expect(sourceChoices({ lockedTo: null, needsGithub: true })[0]).toEqual({
      kind: "git",
      action: "connect-github",
      title: "Connect GitHub",
      description: "Install the Unkey GitHub app to import a repository",
    });
  });

  it("locks the other source once the app has one", () => {
    expect(sourceChoices({ lockedTo: "git", needsGithub: true }).map((c) => c.action)).toEqual([
      "connect-github",
      "locked",
    ]);
  });
});
