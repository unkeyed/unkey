import { z } from "zod";
// clickhouse DateTime returns a string, which we need to parse
export const dateTimeToUnix = z.string().transform((t) => new Date(t).getTime());

export function escapeLikePattern(value: string): string {
  return value.replaceAll("\\", "\\\\").replaceAll("%", "\\%").replaceAll("_", "\\_");
}
