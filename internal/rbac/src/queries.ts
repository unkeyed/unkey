import { z } from "zod";
import { unkeyPermissionValidation } from "./permissions";

type Rule = "and" | "or";

export type RoleQuery<R extends string = string> = {
  version: 1;
  query: NestedQuery<R>;
};

export type NestedQuery<R extends string = string> =
  | R
  | {
      and: NestedQuery<R>[];
      or?: never;
    }
  | {
      and?: never;
      or: NestedQuery<R>[];
    };

export const permissionQuerySchema: z.ZodType<NestedQuery> = z.union([
  unkeyPermissionValidation,
  z.object({
    and: z.array(z.lazy(() => permissionQuerySchema)),
  }),
  z.object({
    or: z.array(z.lazy(() => permissionQuerySchema)),
  }),
]);

type Operation = (...args: NestedQuery[]) => NestedQuery;

const merge = (rule: Rule, ...args: NestedQuery[]): NestedQuery => {
  return args.reduce((acc: NestedQuery, arg) => {
    if (typeof acc === "string") {
      throw new Error("Cannot merge into a string");
    }
    if (!acc[rule]) {
      acc[rule] = [];
    }
    acc[rule]!.push(arg);
    return acc;
  }, {} as NestedQuery);
};

export const or: Operation = (...args) => merge("or", ...args);
export const and: Operation = (...args) => merge("and", ...args);

export function buildQuery(
  fn: (ops: { or: typeof or; and: typeof and }) => NestedQuery,
): RoleQuery {
  return {
    version: 1,
    query: fn({ or, and }),
  };
}
