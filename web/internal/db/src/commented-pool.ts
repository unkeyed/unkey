import { AsyncLocalStorage } from "node:async_hooks";
import type { Pool, PoolConnection, PoolOptions } from "mysql2/promise";
import mysql from "mysql2/promise";
import { type SqlCommentDynamicTags, type SqlCommentStaticTags, annotateSql } from "./sqlcomment";

type Queryable = Pick<PoolConnection, "query" | "execute">;

function wrapQueryable<T extends Queryable>(target: T, staticTags: SqlCommentStaticTags): T {
  const annotateArg = (sql: unknown): unknown => {
    if (typeof sql !== "string") {
      return sql;
    }
    return annotateSql(sql, staticTags, dynamicTagsFromStore());
  };

  return new Proxy(target, {
    get(obj, property, receiver) {
      if (property === "query" || property === "execute") {
        return (sql: unknown, ...args: unknown[]) => {
          const annotated = annotateArg(sql);
          return Reflect.apply(obj[property as "query" | "execute"], obj, [annotated, ...args]);
        };
      }
      return Reflect.get(obj, property, receiver);
    },
  });
}

const dynamicStore = new AsyncLocalStorage<SqlCommentDynamicTags>();

export function runWithSqlCommentTags<T>(tags: SqlCommentDynamicTags, fn: () => T): T {
  return dynamicStore.run(tags, fn);
}

export function dynamicTagsFromStore(): SqlCommentDynamicTags {
  return dynamicStore.getStore() ?? {};
}

export function createCommentedPool(options: PoolOptions, staticTags: SqlCommentStaticTags): Pool {
  const pool = mysql.createPool(options);
  const wrapped = wrapQueryable(pool, staticTags);

  return new Proxy(wrapped, {
    get(target, property, receiver) {
      if (property === "getConnection") {
        return async (...args: unknown[]) => {
          const conn = await pool.getConnection(...(args as []));
          return wrapQueryable(conn, staticTags);
        };
      }
      return Reflect.get(target, property, receiver);
    },
  }) as Pool;
}
