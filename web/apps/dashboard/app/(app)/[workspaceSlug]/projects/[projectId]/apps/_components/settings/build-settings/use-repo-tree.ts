import {
  useAppId,
  useProjectData,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

type ValidationResult = "valid" | "invalid" | "unknown";

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

const workspaceWatchFiles = new Set([
  "bun.lock",
  "bun.lockb",
  "cargo.lock",
  "go.work",
  "go.work.sum",
  "nx.json",
  "package-lock.json",
  "package.json",
  "pnpm-lock.yaml",
  "pnpm-workspace.yaml",
  "turbo.json",
  "yarn.lock",
]);

/** Strip leading "./", leading/trailing slashes so `./svc/api/` and `svc/api` match the same tree entry. */
function normalizePath(path: string): string {
  return path.replace(/^(\.\/)+/, "").replace(/^\/+|\/+$/g, "");
}

/**
 * Join a docker context (root directory) with a relative path.
 * e.g. ("svc/api", "Dockerfile") → "svc/api/Dockerfile"
 *      (".", "Dockerfile")       → "Dockerfile"
 */
function resolveAgainstContext(dockerContext: string, relativePath: string): string {
  const ctx = normalizePath(dockerContext);
  const rel = normalizePath(relativePath);
  if (!ctx || ctx === ".") {
    return rel;
  }
  return `${ctx}/${rel}`;
}

export function useRepoTree() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const { data, isLoading, isError } = trpc.github.getRepoTree.useQuery(
    { projectId, appId },
    { staleTime: 5 * 60 * 1000 },
  );
  const tree = data?.tree ?? null;
  const branch = data?.branch ?? null;
  const isReady = !isLoading && !isError && tree !== null;

  const treeSet = useMemo(() => {
    if (!tree) {
      return null;
    }
    const set = new Map<string, string>();
    for (const entry of tree) {
      set.set(`${entry.type}:${entry.path}`, entry.path);
      set.set(`${entry.type}:${entry.path.toLowerCase()}`, entry.path);
    }
    return set;
  }, [tree]);

  const rootDirectorySuggestions = useMemo(() => {
    if (!tree) {
      return [{ path: ".", marker: "Repository root" }];
    }

    // Suggest likely app roots instead of rendering every directory in a large monorepo.
    // Any repository-relative path can still be entered manually.
    const suggestions = new Map<string, string>();
    for (const entry of tree) {
      if (entry.type !== "blob") {
        continue;
      }

      const fileName = entry.path.split("/").pop() ?? "";
      const normalizedFileName = fileName.toLowerCase();
      if (
        !rootDirectoryMarkers.has(normalizedFileName) &&
        !normalizedFileName.includes("dockerfile")
      ) {
        continue;
      }

      const separatorIndex = entry.path.lastIndexOf("/");
      const path = separatorIndex === -1 ? "." : entry.path.slice(0, separatorIndex);
      if (!suggestions.has(path)) {
        suggestions.set(path, fileName);
      }
    }

    return [
      { path: ".", marker: "Repository root" },
      ...Array.from(suggestions, ([path, marker]) => ({ path, marker }))
        .filter((suggestion) => suggestion.path !== ".")
        .sort((a, b) => a.path.localeCompare(b.path)),
    ];
  }, [tree]);

  const watchPathSuggestions = useMemo(() => {
    if (!tree) {
      return [];
    }

    const suggestions = rootDirectorySuggestions
      .filter((suggestion) => suggestion.path !== ".")
      .map((suggestion) => ({
        path: `${suggestion.path}/**`,
        marker: suggestion.marker,
      }));

    for (const entry of tree) {
      if (
        entry.type === "blob" &&
        !entry.path.includes("/") &&
        workspaceWatchFiles.has(entry.path.toLowerCase())
      ) {
        suggestions.push({ path: entry.path, marker: "Workspace file" });
      }
    }

    return suggestions;
  }, [rootDirectorySuggestions, tree]);

  function validatePath(path: string, type: "blob" | "tree"): ValidationResult {
    if (!isReady || !treeSet) {
      return "unknown";
    }
    const normalized = normalizePath(path);
    if (type === "tree" && (normalized === "." || normalized === "")) {
      return "valid";
    }
    return treeSet.has(`${type}:${normalized}`) ? "valid" : "invalid";
  }

  function findCaseInsensitiveMatch(path: string, type: "blob" | "tree"): string | null {
    if (!treeSet) {
      return null;
    }
    const normalized = normalizePath(path);
    const key = `${type}:${normalized.toLowerCase()}`;
    const match = treeSet.get(key);
    if (match && match !== normalized) {
      return match;
    }
    return null;
  }

  /**
   * Validate a Dockerfile path that is relative to the given docker context.
   * Resolves the full repo path before checking the tree.
   */
  function validateDockerfilePath(dockerfilePath: string, dockerContext: string): ValidationResult {
    const fullPath = resolveAgainstContext(dockerContext, dockerfilePath);
    return validatePath(fullPath, "blob");
  }

  /**
   * Find a case-insensitive match for a Dockerfile path relative to the docker context.
   * Returns the corrected *relative* path (not the full repo path).
   */
  function findDockerfileCaseMatch(dockerfilePath: string, dockerContext: string): string | null {
    const fullPath = resolveAgainstContext(dockerContext, dockerfilePath);
    const match = findCaseInsensitiveMatch(fullPath, "blob");
    if (!match) {
      return null;
    }
    // Strip the context prefix to return a relative path
    const ctx = normalizePath(dockerContext);
    if (ctx && ctx !== "." && match.startsWith(`${ctx}/`)) {
      return match.slice(ctx.length + 1);
    }
    return match;
  }

  /**
   * Get all Dockerfiles in the repo, returned as paths relative to the given docker context.
   * Only includes Dockerfiles that are under the context directory.
   */
  function getDockerfilesForContext(dockerContext: string): string[] {
    if (!tree) {
      return [];
    }
    const ctx = normalizePath(dockerContext);
    return tree
      .filter((entry) => {
        const fileName = entry.path.split("/").pop() ?? "";
        if (entry.type !== "blob" || !fileName.toLowerCase().includes("dockerfile")) {
          return false;
        }
        if (!ctx || ctx === ".") {
          return true;
        }
        return entry.path.startsWith(`${ctx}/`);
      })
      .map((entry) => {
        if (!ctx || ctx === ".") {
          return entry.path;
        }
        return entry.path.slice(ctx.length + 1);
      });
  }

  return {
    branch,
    validatePath,
    findCaseInsensitiveMatch,
    rootDirectorySuggestions,
    watchPathSuggestions,
    validateDockerfilePath,
    findDockerfileCaseMatch,
    getDockerfilesForContext,
  };
}
