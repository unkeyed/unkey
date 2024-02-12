import { z } from "zod";
import { unkeyPermissionValidation } from "./permissions";

type Rule = "and" | "or";

export type PermissionQuery<R extends string = string> = {
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

function merge<R extends string>(rule: Rule, ...args: NestedQuery<R>[]): NestedQuery<R> {
  return args.reduce(
    (acc: NestedQuery<R>, arg) => {
      if (typeof acc === "string") {
        throw new Error("Cannot merge into a string");
      }
      if (!acc[rule]) {
        acc[rule] = [];
      }
      acc[rule]!.push(arg);
      return acc;
    },
    {} as NestedQuery<R>,
  );
}

export function or<R extends string = string>(...args: NestedQuery<R>[]): NestedQuery<R> {
  return merge("or", ...args);
}

export function and<R extends string = string>(...args: NestedQuery<R>[]): NestedQuery<R> {
  return merge("and", ...args);
}
export function buildQuery<R extends string = string>(
  fn: (ops: { or: typeof or<R>; and: typeof and<R> }) => NestedQuery<R>,
): PermissionQuery {
  return {
    version: 1,
    query: fn({ or, and }),
  };
}

/**
 * buildUnkeyQuery is preloaded with out available roles and ensures typesafety for root key validation
 */
export const buildUnkeyQuery = buildQuery<z.infer<typeof unkeyPermissionValidation>>;
