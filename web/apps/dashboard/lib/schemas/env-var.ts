import { z } from "zod";

// Environment variables exist to become process env: builds expose them to
// install/build commands and the runtime injects them into the container.
// POSIX shell names are the only names every consumer can actually read, so
// anything else is rejected at creation. Keep in sync with
// pkg/validation/env_var.go.
export const envVarKeySchema = z
  .string()
  .trim()
  .min(1, "Variable name is required")
  .regex(
    /^[A-Za-z_][A-Za-z0-9_]*$/,
    "Only letters, digits, and underscores are allowed, and the name must not start with a digit",
  );

export const envVarValueSchema = z
  .string()
  .trim()
  .min(1, "Variable value is required")
  .refine(
    (val) => !val.includes("\\n") && !val.includes("\\r"),
    "Newline characters are not allowed",
  );
