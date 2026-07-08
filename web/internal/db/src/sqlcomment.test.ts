import { describe, expect, it } from "vitest";
import { dynamicTagsFromStore, runWithSqlCommentTags } from "./commented-pool";
import { annotateSql, staticTagsFromEnv } from "./sqlcomment";

const dashboardTags = {
  application: "unkey",
  service: "dashboard",
  region: "us-east-1",
  releaseSha: "a1b2c3d",
};

// Representative Drizzle/mysql2 SQL (backtick-quoted identifiers, positional params).
const drizzleSelectKeys =
  "select `keys`.`id`, `keys`.`name` from `keys` where `keys`.`workspace_id` = ? limit ?";

const drizzleInsertEnvVar =
  "insert into `app_environment_variables` (`id`, `app_id`, `key`, `value`) values (?, ?, ?, ?)";

describe("sqlcomment", () => {
  it("leaves SQL unchanged when tagging is disabled", () => {
    expect(annotateSql(drizzleSelectKeys, { application: "unkey", service: "" })).toBe(
      drizzleSelectKeys,
    );
  });

  it("tags drizzle select queries with static service metadata", () => {
    const got = annotateSql(drizzleSelectKeys, dashboardTags);

    expect(got).toContain(drizzleSelectKeys);
    expect(got).toContain("service='dashboard'");
    expect(got).toContain("release_sha='a1b2c3d'");
    expect(got).not.toContain("mode=");
  });

  it("tags drizzle writes with tRPC route and source from async context", () => {
    const got = runWithSqlCommentTags({ route: "deploy.envVars.create", source: "trpc" }, () =>
      annotateSql(drizzleInsertEnvVar, dashboardTags, dynamicTagsFromStore()),
    );

    expect(got).toContain(drizzleInsertEnvVar);
    expect(got).toContain("route='deploy.envVars.create'");
    expect(got).toContain("source='trpc'");
    expect(got).not.toContain("mode=");
  });

  it("omits route and source when no request context is set", () => {
    const got = annotateSql(drizzleSelectKeys, dashboardTags);

    expect(got).not.toContain("route=");
    expect(got).not.toContain("source=");
  });

  it("meta-escapes single quotes after url-encoding", () => {
    const got = annotateSql(drizzleSelectKeys, dashboardTags, {
      route: "keys.o'brien",
      source: "trpc",
    });

    expect(got).toContain(`route='keys.o\\'brien'`);
  });

  it("url-encodes spaces and slashes in route values", () => {
    const got = annotateSql(drizzleSelectKeys, dashboardTags, {
      route: "POST /v2/keys.verifyKey",
      source: "http",
    });

    expect(got).toContain(`route='POST%20%2Fv2%2Fkeys.verifyKey'`);
  });

  it("builds static tags from env", () => {
    const previous = process.env.UNKEY_GIT_COMMIT_SHA;
    process.env.UNKEY_GIT_COMMIT_SHA = "abcdef012345";
    try {
      expect(staticTagsFromEnv("portal").releaseSha).toBe("abcdef0");
    } finally {
      process.env.UNKEY_GIT_COMMIT_SHA = previous;
    }
  });
});
