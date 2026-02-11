/**
 * Parses projectId from where expression.
 *
 * Structure:
 * - Direct: { name: "eq", type: "func", args: [{ path: ["projectId"], type: "ref" }, { value: "proj_xxx", type: "val" }] }
 * - And: { name: "and", type: "func", args: [eq(...), eq(...)] }
 */
export function parseProjectIdFromWhere(where?: any): string | null {
  if (!where) {
    return null;
  }

  // Helper to check if an expression is eq(projectId, value)
  function isProjectIdEq(expr: any): string | null {
    if (expr?.name !== "eq" || expr?.type !== "func" || !Array.isArray(expr?.args)) {
      return null;
    }

    const [fieldRef, valueRef] = expr.args;

    // Check if first arg is { path: ["projectId"], type: "ref" }
    if (
      fieldRef?.type === "ref" &&
      Array.isArray(fieldRef?.path) &&
      fieldRef.path.length === 1 &&
      fieldRef.path[0] === "projectId"
    ) {
      // Second arg is { value: "proj_xxx", type: "val" }
      if (valueRef?.type === "val" && typeof valueRef?.value === "string") {
        return valueRef.value;
      }
    }

    return null;
  }

  // Check direct eq(projectId, value)
  const directMatch = isProjectIdEq(where);
  if (directMatch) {
    return directMatch;
  }

  // Check and(eq(projectId, value), ...)
  if (where?.name === "and" && where?.type === "func" && Array.isArray(where?.args)) {
    for (const arg of where.args) {
      const match = isProjectIdEq(arg);
      if (match) {
        return match;
      }
    }
  }

  return null;
}

/**
 * Runtime dev-mode validator for collections.
 * Throws helpful error if projectId filter is missing.
 * Only active in development mode (process.env.NODE_ENV !== 'production').
 */
export function validateProjectIdInQuery(where?: any): void {
  if (process.env.NODE_ENV === "production") {
    return;
  }

  if (!where) {
    throw new Error(
      "Deploy collections require projectId filter: .where(({ collection }) => eq(collection.projectId, projectId))",
    );
  }

  const projectId = parseProjectIdFromWhere(where);
  if (!projectId) {
    throw new Error(
      "Deploy collections require projectId as first constraint: .where(({ collection }) => eq(collection.projectId, projectId))",
    );
  }
}
