import { z } from "zod";

export const roleQuery = z.union([
  z.object({ role: z.string() }),
  z.object({
    and: z.union([
      z.object({ role: z.string() }),
      z.object({
        and: z.union([
          z.object({ role: z.string() }),
          z.object({
            and: z.array(z.string()),
          }),
          z.object({
            or: z.array(z.string()),
          }),
        ]),
      }),
      z.object({
        or: z.union([
          z.object({ role: z.string() }),
          z.object({
            and: z.array(z.string()),
          }),
          z.object({
            or: z.array(z.string()),
          }),
        ]),
      }),
    ]),
  }),
  z.object({
    or: z.union([
      z.object({ role: z.string() }),
      z.object({
        and: z.union([
          z.object({ role: z.string() }),
          z.object({
            and: z.array(z.string()),
          }),
          z.object({
            or: z.array(z.string()),
          }),
        ]),
      }),
      z.object({
        or: z.union([
          z.object({ role: z.string() }),
          z.object({
            and: z.array(z.string()),
          }),
          z.object({
            or: z.array(z.string()),
          }),
        ]),
      }),
    ]),
  }),
]);

type Entry = z.infer<typeof roleQuery>;

type Operation = (...args: (string | Entry)[]) => Entry;

const merge = (op: "and" | "or", ...args: Entry[]): Entry => {
  return args.reduce((acc: Entry, arg) => {
    if ("role" in acc) {
      throw new Error("Cannot merge into a role");
    }
    if (!acc[op]) {
      acc[op] = [];
    }
    acc[op]!.push(arg);
    return acc;
  }, {} as Entry);
};

export const or: Operation = (...args) => merge("or", ...args);
export const and: Operation = (...args) => merge("and", ...args);

const x = and("abc", or("def", "ghi", and("jkl", "mno")));

console.log(JSON.stringify(x, null, 2));
