import { z } from "zod";

export type RepoTreeEntry = { path: string; type: string };

const dockerContextSegment = /^[A-Za-z0-9._-]+$/;

export const dockerContextSchema = z
  .string()
  .min(1, "Enter a root directory or use '.' for the repository root.")
  .refine(
    (path) =>
      path === "." ||
      (path === path.trim() &&
        !path.startsWith("/") &&
        !path.includes("\\") &&
        path
          .split("/")
          .every(
            (segment) => segment !== "." && segment !== ".." && dockerContextSegment.test(segment),
          )),
    "Enter a path relative to the repository root, like api or services/api.",
  );

const rootDirectoryMarkers = new Set([
  "build.gradle",
  "build.gradle.kts",
  "cargo.toml",
  "composer.json",
  "gemfile",
  "go.mod",
  "mix.exs",
  "package.json",
  "pipfile",
  "pom.xml",
  "pyproject.toml",
  "requirements.txt",
]);

export function normalizePath(path: string): string {
  return path.replace(/^(\.\/)+/, "").replace(/^\/+|\/+$/g, "");
}

export type RootDirectorySuggestion = { path: string; marker: string };

export function suggestRootDirectories(tree: readonly RepoTreeEntry[]): RootDirectorySuggestion[] {
  const markersByPath = new Map<string, string>();
  for (const entry of tree) {
    if (entry.type !== "blob") {
      continue;
    }
    const fileName = entry.path.split("/").pop() ?? "";
    const normalized = fileName.toLowerCase();
    if (!rootDirectoryMarkers.has(normalized) && !normalized.includes("dockerfile")) {
      continue;
    }
    const separatorIndex = entry.path.lastIndexOf("/");
    const path = separatorIndex === -1 ? "." : entry.path.slice(0, separatorIndex);
    if (path !== "." && !markersByPath.has(path)) {
      markersByPath.set(path, fileName);
    }
  }

  return [
    { path: ".", marker: "Repository root" },
    ...Array.from(markersByPath, ([path, marker]) => ({ path, marker })).sort((a, b) =>
      a.path.localeCompare(b.path),
    ),
  ];
}

export function findDockerfiles(tree: readonly RepoTreeEntry[], dockerContext: string): string[] {
  const context = normalizePath(dockerContext);
  const prefix = context === "" || context === "." ? "" : `${context}/`;
  return tree
    .filter((entry) => {
      const fileName = entry.path.split("/").pop() ?? "";
      return (
        entry.type === "blob" &&
        fileName.toLowerCase().includes("dockerfile") &&
        entry.path.startsWith(prefix)
      );
    })
    .map((entry) => entry.path.slice(prefix.length));
}
