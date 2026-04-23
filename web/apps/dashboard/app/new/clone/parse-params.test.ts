import { describe, expect, it } from "vitest";
import { buildCloneSearchString, parseCloneParams } from "./parse-params";

describe("parseCloneParams", () => {
  it("rejects missing repository", () => {
    const result = parseCloneParams({});
    expect(result.ok).toBe(false);
  });

  it("parses a full GitHub URL", () => {
    const result = parseCloneParams({
      repository: "https://github.com/acme/widgets",
    });
    expect(result.ok).toBe(true);
    if (!result.ok) {
      return;
    }
    expect(result.params.repository).toEqual({
      owner: "acme",
      repo: "widgets",
      fullName: "acme/widgets",
      url: "https://github.com/acme/widgets",
    });
  });

  it("parses a GitHub URL with .git suffix and trailing slash", () => {
    const result = parseCloneParams({
      repository: "https://github.com/acme/widgets.git/",
    });
    expect(result.ok).toBe(true);
    if (!result.ok) {
      return;
    }
    expect(result.params.repository.fullName).toBe("acme/widgets");
  });

  it("parses owner/repo shorthand", () => {
    const result = parseCloneParams({ repository: "acme/widgets" });
    expect(result.ok).toBe(true);
    if (!result.ok) {
      return;
    }
    expect(result.params.repository.url).toBe("https://github.com/acme/widgets");
  });

  it("rejects a garbage repository value", () => {
    const result = parseCloneParams({ repository: "not a repo" });
    expect(result.ok).toBe(false);
  });

  it("defaults optional fields to null", () => {
    const result = parseCloneParams({ repository: "acme/widgets" });
    if (!result.ok) {
      throw new Error("expected ok");
    }
    expect(result.params.projectName).toBeNull();
    expect(result.params.branch).toBeNull();
    expect(result.params.rootDirectory).toBeNull();
    expect(result.params.dockerfile).toBeNull();
    expect(result.params.envDescription).toBeNull();
    expect(result.params.envLink).toBeNull();
    expect(result.params.envKeys).toEqual([]);
  });

  it("parses env keys and dedups while preserving order", () => {
    const result = parseCloneParams({
      repository: "acme/widgets",
      env: "FOO,BAR,FOO, BAZ ,",
    });
    if (!result.ok) {
      throw new Error("expected ok");
    }
    expect(result.params.envKeys).toEqual(["FOO", "BAR", "BAZ"]);
  });

  it("rejects invalid env key characters", () => {
    const result = parseCloneParams({
      repository: "acme/widgets",
      env: "OK,BAD KEY",
    });
    expect(result.ok).toBe(false);
  });

  it("rejects a non-URL envLink", () => {
    const result = parseCloneParams({
      repository: "acme/widgets",
      envLink: "not-a-url",
    });
    expect(result.ok).toBe(false);
  });

  it("uses the first value when a query param repeats", () => {
    const result = parseCloneParams({
      repository: ["acme/widgets", "other/repo"],
    });
    if (!result.ok) {
      throw new Error("expected ok");
    }
    expect(result.params.repository.fullName).toBe("acme/widgets");
  });

  it("round-trips through buildCloneSearchString", () => {
    const first = parseCloneParams({
      repository: "https://github.com/acme/widgets",
      "project-name": "Widgets",
      branch: "main",
      "root-directory": "apps/api",
      dockerfile: "docker/api.Dockerfile",
      env: "FOO,BAR",
      envDescription: "API secrets",
      envLink: "https://acme.com/docs",
    });
    if (!first.ok) {
      throw new Error("expected ok");
    }

    const rebuilt = parseCloneParams(
      Object.fromEntries(new URLSearchParams(buildCloneSearchString(first.params))),
    );
    if (!rebuilt.ok) {
      throw new Error("expected rebuilt ok");
    }
    expect(rebuilt.params).toEqual(first.params);
  });
});
