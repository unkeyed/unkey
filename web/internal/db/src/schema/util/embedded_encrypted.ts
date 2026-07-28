import { caseSensitiveVarchar } from "./case_sensitive_varchar";

export const embeddedEncrypted = {
  /**
   * The encrypted base64 encoded response from vault
   * Store it as is and send it back to the vault for decryption as is.
   */
  encrypted: caseSensitiveVarchar("encrypted", { length: 1024 }).notNull(),

  /**
   * An identifier for the key used to encrypt this. Useful for knowing what keys are still being used.
   */
  encryptionKeyId: caseSensitiveVarchar("encryption_key_id", { length: 256 }).notNull(),
};
