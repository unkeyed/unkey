import { newId } from "@unkey/id";
import { MySqlDialect } from "drizzle-orm/mysql-core";
import { expect, it } from "vitest";
import { insertApiQuery } from "./api";

it("creates APIs without depending on the retired IP whitelist column", () => {
  const api = {
    id: newId("api"),
    name: "Customer's API",
    workspaceId: newId("workspace"),
    projectId: newId("project"),
    keyAuthId: newId("keyAuth"),
    authType: "key" as const,
    createdAtM: 1_798_000_000_123,
  };
  const query = new MySqlDialect().sqlToQuery(insertApiQuery(api));

  expect(query.sql).not.toContain("ip_whitelist");
  expect(query.sql).not.toContain(api.name);
  expect(query.params).toEqual(expect.arrayContaining(Object.values(api)));
});
