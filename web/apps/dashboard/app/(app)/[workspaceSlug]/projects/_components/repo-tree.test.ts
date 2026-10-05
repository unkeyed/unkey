import { describe, expect, it } from "vitest";
import { dockerContextSchema, findDockerfiles, suggestRootDirectories } from "./repo-tree";

const tree = [
  { path: "package.json", type: "blob" },
  { path: "apps", type: "tree" },
  { path: "apps/web/package.json", type: "blob" },
  { path: "apps/web/Dockerfile", type: "blob" },
  { path: "services/api/go.mod", type: "blob" },
  { path: "services/api/main.go", type: "blob" },
  { path: "Dockerfile.dev", type: "blob" },
];

describe("dockerContextSchema", () => {
  it.each([".", "api", "services/api"])("accepts root directory %s", (path) => {
    expect(dockerContextSchema.safeParse(path).success).toBe(true);
  });

  it.each(["", "/api", "./api", "api/../web", "api\\web", " api"])(
    "rejects root directory %s",
    (path) => {
      expect(dockerContextSchema.safeParse(path).success).toBe(false);
    },
  );
});

describe("suggestRootDirectories", () => {
  it("lists the root first, then directories that hold a project marker", () => {
    expect(suggestRootDirectories(tree)).toEqual([
      { path: ".", marker: "Repository root" },
      { path: "apps/web", marker: "package.json" },
      { path: "services/api", marker: "go.mod" },
    ]);
  });

  it("offers only the root for an empty tree", () => {
    expect(suggestRootDirectories([])).toEqual([{ path: ".", marker: "Repository root" }]);
  });
});

describe("findDockerfiles", () => {
  it("lists Dockerfiles relative to the root directory", () => {
    expect(findDockerfiles(tree, ".")).toEqual(["apps/web/Dockerfile", "Dockerfile.dev"]);
    expect(findDockerfiles(tree, "apps/web")).toEqual(["Dockerfile"]);
    expect(findDockerfiles(tree, "services/api")).toEqual([]);
  });

  it("reads an empty or slash-wrapped root directory like the repository paths", () => {
    expect(findDockerfiles(tree, "")).toEqual(["apps/web/Dockerfile", "Dockerfile.dev"]);
    expect(findDockerfiles(tree, "./apps/web/")).toEqual(["Dockerfile"]);
  });
});
