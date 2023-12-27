import { z } from "zod";

import { type Result, result } from "@unkey/result";
type Rule = "and" | "or";

export type RoleQuery =
  | string
  | {
      and: RoleQuery[];
      or?: never;
    }
  | {
      and?: never;
      or: RoleQuery[];
    };

export const roleQuerySchema: z.ZodType<RoleQuery> = z.union([
  z.string(),
  z.object({
    and: z.array(z.lazy(() => roleQuerySchema)),
  }),
  z.object({
    or: z.array(z.lazy(() => roleQuerySchema)),
  }),
]);

type Operation = (...args: RoleQuery[]) => RoleQuery;

const merge = (rule: Rule, ...args: RoleQuery[]): RoleQuery => {
  return args.reduce((acc: RoleQuery, arg) => {
    if (typeof acc === "string") {
      throw new Error("Cannot merge into a string");
    }
    if (!acc[rule]) {
      acc[rule] = [];
    }
    acc[rule]!.push(arg);
    return acc;
  }, {} as RoleQuery);
};

export const or: Operation = (...args) => merge("or", ...args);
export const and: Operation = (...args) => merge("and", ...args);

export const evaluateRoles = (
  query: RoleQuery,
  roles: string[],
): Result<{ valid: true } | { valid: false; message: string }> => {
  if (roles.length === 0) {
    return result.fail({ message: "No roles provided" });
  }

  if (typeof query === "string") {
    // Check if the role is in the list of roles
    if (roles.includes(query)) {
      return result.success({ valid: true });
    } else {
      return result.success({ valid: false, message: `Role ${query} not allowed` });
    }
  }

  if (query.and) {
    const results = query.and.map((q) => evaluateRoles(q, roles));
    for (const r of results) {
      if (r.error) {
        return r;
      }
      if (!r.value.valid) {
        return r;
      }
    }
    return result.success({ valid: true });
  }

  if (query.or) {
    for (const q of query.or) {
      const r = evaluateRoles(q, roles);
      if (r.error) {
        return r;
      }
      if (r.value.valid) {
        return r;
      }
    }
    return result.success({ valid: false, message: "No role matched" });
  }

  return result.fail({ message: "reached end of evaluate and no match" });
};
