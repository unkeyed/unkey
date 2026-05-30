import { z } from "zod";

// Keys must be valid environment variable names. Dots and hyphens are rejected
// because they are not valid shell identifiers (the build sources the vars via
// `. /run/secrets/.env`) and k8s envFrom silently drops them at runtime, so
// such keys would never reach the container.
export const envVarKeySchema = z
  .string()
  .trim()
  .min(1, "Variable name is required")
  .regex(
    /^[A-Za-z_][A-Za-z0-9_]*$/,
    "Must start with a letter or underscore and contain only letters, numbers, and underscores",
  );

// Values may span multiple lines (PEM keys, JSON blobs). The build pipeline
// single-quotes them into the .env secret so `set -a && . /run/secrets/.env`
// loads them intact. 4096 matches the `value` column (varchar(4096)).
export const envVarValueSchema = z
  .string()
  .trim()
  .min(1, "Variable value is required")
  .max(4096, "Variable value must be at most 4096 characters");
