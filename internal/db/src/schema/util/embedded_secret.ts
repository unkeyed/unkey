import { int, mysqlEnum, varchar } from "drizzle-orm/mysql-core";

/**
 * Embed all required columns to store a secret
 *
 * @example
 * ```ts
 * export const myTable = {
 * ...emebddedSecret,
 * id: ...
 *
 * }
 *
 * ```
 */
export const embeddedSecret = {
  algorithm: mysqlEnum("algorithm", ["AES-GCM"]).notNull(),
  iv: varchar("iv", { length: 255 }).notNull(),
  ciphertext: varchar("ciphertext", { length: 1024 }).notNull(),
  encryptionKeyVersion: int("encryption_key_version").notNull().default(1),
};
