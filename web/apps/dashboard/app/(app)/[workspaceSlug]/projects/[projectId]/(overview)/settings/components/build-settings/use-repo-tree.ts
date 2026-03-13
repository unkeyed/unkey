import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { useProjectData } from "../../../data-provider";

type ValidationResult = "valid" | "invalid" | "unknown";

/** Strip leading/trailing slashes so `/svc/api/` and `svc/api` match the same tree entry. */
function normalizePath(path: string): string {
  return path.replace(/^\/+|\/+$/g, "");
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
  const { data, isLoading, isError } = trpc.github.getRepoTree.useQuery(
    { projectId },
    { staleTime: 5 * 60 * 1000 },
  );

  const tree = data?.tree ?? null;
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
        if (entry.type !== "blob" || entry.path.split("/").pop() !== "Dockerfile") {
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
    validatePath,
    findCaseInsensitiveMatch,
    validateDockerfilePath,
    findDockerfileCaseMatch,
    getDockerfilesForContext,
  };
}
