import { sql } from "@unkey/db";
import { z } from "zod";

const flagValues = z.record(
  z.string(),
  z.union([z.literal(0), z.literal(1)]).transform((value) => value === 1),
);

export function workspaceFlagExtras() {
  return {
    flags: sql`(
      SELECT COALESCE(JSON_OBJECTAGG(f.slug, COALESCE(o.value, f.default_value)), JSON_OBJECT())
      FROM flags AS f
      LEFT JOIN workspace_flag_overrides AS o
        ON o.flag_id = f.id AND o.workspace_id = ${sql.identifier("workspaces")}.${sql.identifier("id")}
    )`
      .mapWith((value: unknown) =>
        flagValues.parse(typeof value === "string" ? JSON.parse(value) : value),
      )
      .as("flags"),
  };
}
