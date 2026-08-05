import { bigint } from "drizzle-orm/mysql-core";

export function primaryKey() {
  return bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey();
}
