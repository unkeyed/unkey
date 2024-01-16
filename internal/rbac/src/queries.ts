import { z } from "zod";

type Rule = "and" | "or";

export type RoleQuery = {
  version: 1;
  query: NestedQuery;
};

export type NestedQuery =
  | string
  | {
      and: NestedQuery[];
      or?: never;
    }
  | {
      and?: never;
      or: NestedQuery[];
    };

export const roleQuerySchema: z.ZodType<NestedQuery> = z.union([
  z.string(),
  z.object({
    and: z.array(z.lazy(() => roleQuerySchema)),
  }),
  z.object({
    or: z.array(z.lazy(() => roleQuerySchema)),
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
