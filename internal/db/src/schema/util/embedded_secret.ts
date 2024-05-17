import { int, mysqlEnum, varchar } from "drizzle-orm/mysql-core";

export const embeddedSecret = {
  /**
   * The algorithm used to encrypt
   */
  algorithm: mysqlEnum("algorithm", ["AES-GCM"]).notNull(),

  /**
   * The AES family of algorithms require an initialization vector for the ciphers.
   * This is usually a random 32 byte vector
   */
  iv: varchar("iv", { length: 255 }).notNull(),

  /**
   * The encrypted data encoded as base64 string
   */
  ciphertext: varchar("ciphertext", { length: 1024 }).notNull(),

  /**
   * The encryption key version used for this secret.
   *
   * We annotate each key with a version to make migrations possible.
   */
  keyVersion: int("key_version").notNull().default(1),
};
