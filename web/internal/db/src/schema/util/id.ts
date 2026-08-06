import { customType } from "drizzle-orm/mysql-core";

export const id = customType<{
  data: string;
}>({
  dataType() {
    return "varchar(48) COLLATE utf8mb4_0900_as_cs";
  },
});
