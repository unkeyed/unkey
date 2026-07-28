import { customType } from "drizzle-orm/mysql-core";

export const caseSensitiveVarchar = customType<{
  data: string;
  config: { length: number };
  configRequired: true;
}>({
  dataType(config) {
    return `varchar(${config.length}) COLLATE utf8mb4_0900_as_cs`;
  },
});
