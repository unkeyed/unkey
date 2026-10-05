import { describe, expect, it } from "vitest";
import { resolveCrashHelp } from "./crash-help-state";

const automatic = { dockerfile: "", dockerContext: ".", port: 8080, command: [] };
const tree = [
  { path: "app/Dockerfile", type: "blob" },
  { path: "package.json", type: "blob" },
];

describe("resolveCrashHelp", () => {
  it("waits for settings", () => {
    expect(resolveCrashHelp({ settings: null, tree })).toEqual({ type: "loading" });
  });

  it("points at a Dockerfile when the app builds automatically and the repo has one", () => {
    expect(resolveCrashHelp({ settings: automatic, tree })).toEqual({
      type: "dockerfile",
      dockerfile: "app/Dockerfile",
      rootDirectory: ".",
    });
  });

  it("falls back to the checklist from real settings", () => {
    expect(
      resolveCrashHelp({
        settings: { ...automatic, dockerfile: "app/Dockerfile", command: ["node", "server.js"] },
        tree,
      }),
    ).toEqual({
      type: "checklist",
      port: 8080,
      startCommand: "node server.js",
      rootDirectory: ".",
    });
    expect(resolveCrashHelp({ settings: automatic, tree: null })).toEqual({
      type: "checklist",
      port: 8080,
      startCommand: "Automatic",
      rootDirectory: ".",
    });
  });
});
