import { customType } from "drizzle-orm/mysql-core";

/**
 * Custom longblob type for MySQL in Drizzle ORM
 *
 * Usage:
 * ```typescript
 * import { longblob } from "./util/longblob";
 *
 * export const myTable = mysqlTable("my_table", {
 *   data: longblob("data"),
 * });
 * ```
 */
export const longblob = customType<{
  data: string | null;
  driverData: string | null;
}>({
  dataType() {
    return "longblob";
  },
  toDriver(value: string | null): string | null {
    return value;
  },
  fromDriver(value: string | null): string | null {
    return value;
  },
});
