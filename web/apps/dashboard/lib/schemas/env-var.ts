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
// app_environment_variables.value (varchar 4096). Vault base64-encodes a
// protobuf wrapper around the AES-GCM ciphertext, so the encrypted string is
// roughly 4/3 * (plaintext_bytes + 71). Capping the plaintext at 3000 *bytes*
// keeps the encrypted output under the 4096 column limit. The budget is in
// UTF-8 bytes, not characters: a multibyte value (e.g. CJK/emoji) can be far
// larger in bytes than in characters, so .max() on string length would not
// protect the column.
const MAX_ENV_VAR_VALUE_BYTES = 3000;
const utf8Encoder = new TextEncoder();

export const envVarValueSchema = z
  .string()
  .trim()
  .min(1, "Variable value is required")
  .refine(
    (val) => utf8Encoder.encode(val).length <= MAX_ENV_VAR_VALUE_BYTES,
    `Variable value must be at most ${MAX_ENV_VAR_VALUE_BYTES} bytes`,
  )
  // Builds serialize env vars into a line-oriented .env file that the Dockerfile
  // shell-sources, so an embedded newline corrupts the format. Nothing rejects
  // this on write: the v2 setEnvironmentVariables handler encrypts and stores
  // the value unexamined, and buildEnvFileSecret in svc/ctrl/worker/deploy/
  // build.go only sees it at deploy time, where it fails as a non-retryable
  // terminal error. So this is not a client mirror of a server rule: it is the
  // only pre-build guard, and only on the dashboard path. Values written through
  // the API still reach the build unchecked.
  //
  // Matches actual 0x0A/0x0D, not the two-character sequence \n, which is
  // legitimate in Windows paths and regexes. .trim() above already removes
  // leading and trailing newlines, so only embedded ones reach here.
  //
  // The message names replacement because values with real newlines predate this
  // check: an <input> strips CR/LF from display, so in the edit row the rejected
  // characters are invisible and the rule alone would point at a clean-looking
  // field.
  .refine(
    (val) => !/[\n\r]/.test(val),
    "Newline characters are not allowed. Replace this with a single-line value.",
  );
