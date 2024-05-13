import { bigint } from "drizzle-orm/mysql-core";

export const lifecycleDates = {
  createdAt: bigint("created_at", { mode: "number" })
    .notNull()
    .$defaultFn(() => Date.now()),
  updatedAt: bigint("updated_at", { mode: "number" }).$onUpdateFn(() => Date.now()),
};

export const lifecycleDatesMigration = {
  createdAtN: bigint("created_at_n", { mode: "number" })
    .notNull()
    .$defaultFn(() => Date.now()),
  updatedAtN: bigint("updated_at_n", { mode: "number" }).$onUpdateFn(() => Date.now()),
};
