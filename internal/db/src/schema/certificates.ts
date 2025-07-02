import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  text,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";

export const certificates = mysqlTable(
  "certificates",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    // Domain this certificate is for
    hostname: varchar("hostname", { length: 256 }).notNull(),

    // Certificate type
    certificateType: mysqlEnum("certificate_type", [
      "wildcard", // *.unkey.app
      "custom", // Custom domain
      "self_signed", // For development
    ]).notNull(),

    // Certificate data (PEM format)
    certificatePem: text("certificate_pem").notNull(),
    certificateChain: text("certificate_chain"), // Full chain if needed

    // Private key (encrypted)
    privateKeyEncrypted: text("private_key_encrypted").notNull(),
    encryptionKeyId: varchar("encryption_key_id", { length: 256 }).notNull(),

    // Certificate metadata
    issuer: varchar("issuer", { length: 256 }), // Let's Encrypt, AWS ACM, etc.
    serialNumber: varchar("serial_number", { length: 256 }),
    fingerprint: varchar("fingerprint", { length: 128 }), // SHA-256 fingerprint

    // Validity period
    notBefore: bigint("not_before", { mode: "number" }).notNull(), // Unix timestamp
    notAfter: bigint("not_after", { mode: "number" }).notNull(), // Unix timestamp

    // Renewal status
    status: mysqlEnum("status", [
      "active",
      "expiring_soon", // Within renewal window
      "expired",
      "revoked",
      "pending_renewal",
    ])
      .notNull()
      .default("active"),

    // Auto-renewal configuration
    autoRenew: boolean("auto_renew").notNull().default(true),
    renewalAttempts: int("renewal_attempts").default(0),
    lastRenewalAttempt: bigint("last_renewal_attempt", { mode: "number" }),

    // ACME challenge data for Let's Encrypt
    acmeAccountId: varchar("acme_account_id", { length: 256 }),

    ...lifecycleDatesMigration,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    hostnameIdx: uniqueIndex("hostname_idx").on(table.hostname),
    statusIdx: index("status_idx").on(table.status),
    expirationIdx: index("expiration_idx").on(table.notAfter),
    fingerprintIdx: index("fingerprint_idx").on(table.fingerprint),
  }),
);

export const certificatesRelations = relations(certificates, () => ({
  // Relations defined but no foreign keys enforced
  // workspace: one(workspaces),
  // hostnames: many(hostnames),
}));
