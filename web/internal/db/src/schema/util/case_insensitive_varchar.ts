import { customType } from "drizzle-orm/mysql-core";

export const caseInsensitiveVarchar = customType<{
  data: string;
  config: { length: number };
  configRequired: true;
}>({
  dataType(config) {
    return `varchar(${config.length})`;
  },
});
