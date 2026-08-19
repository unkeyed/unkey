import { z } from "zod";

// Environment variables exist to become process env: builds expose them to
// install/build commands and the runtime injects them into the container.
// POSIX shell names are the only names every consumer can actually read, so
// anything else is rejected at creation. Keep in sync with
// pkg/validation/env_var.go.
// Keys are persisted verbatim into app_environment_variables.key (varchar 256).
const MAX_ENV_VAR_KEY_LENGTH = 256;

export const envVarKeySchema = z
  .string()
  .trim()
  .min(1, "Variable name is required")
  .max(MAX_ENV_VAR_KEY_LENGTH, `Variable name must be at most ${MAX_ENV_VAR_KEY_LENGTH} characters`)
  .regex(
    /^[A-Za-z_][A-Za-z0-9_]*$/,
    "Only letters, digits, and underscores are allowed, and the name must not start with a digit",
  );

// Values are encrypted before storage and the ciphertext lands in
// app_environment_variables.value (TEXT, 65,535-byte capacity). Vault
// base64-encodes a protobuf wrapper around the AES-GCM ciphertext, so the
// encrypted string is roughly 4/3 * (plaintext_bytes + 71). Capping the
// plaintext at 16,384 *bytes* keeps the encrypted output near 22 KiB, well
// inside the column, and covers RSA-4096 keys and typical cert chains. The
// budget is in UTF-8 bytes, not characters: a multibyte value (e.g. CJK/emoji)
// can be far larger in bytes than in characters, so .max() on string length
// would not protect the column.
const MAX_ENV_VAR_VALUE_BYTES = 16384;
const utf8Encoder = new TextEncoder();

export const envVarValueSchema = z
  .string()
  // .trim() strips leading and trailing whitespace, including the trailing
  // newline a pasted PEM block usually carries. Interior LF newlines are kept
  // so multi-line values (PEM keys, certs, JSON) round-trip unchanged.
  .trim()
  .min(1, "Variable value is required")
  .refine(
    (val) => utf8Encoder.encode(val).length <= MAX_ENV_VAR_VALUE_BYTES,
    `Variable value must be at most ${MAX_ENV_VAR_VALUE_BYTES} bytes`,
  )
  // Reject carriage returns so stored values are canonical LF. A CRLF value
  // would otherwise store a stray \r that shell sourcing and dotenv parsers
  // treat inconsistently.
  .refine((val) => !val.includes("\r"), "Carriage return characters are not allowed");
