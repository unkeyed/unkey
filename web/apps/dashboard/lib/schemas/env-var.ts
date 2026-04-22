import { z } from "zod";

export const envVarKeySchema = z
  .string()
  .trim()
  .min(1, "Variable name is required")
  .regex(/^[-._a-zA-Z0-9]+$/, "Only letters, numbers, hyphens, underscores, and dots are allowed");

export const envVarValueSchema = z
  .string()
  .min(1, "Variable value is required")
  .refine(
    (val) => !val.includes("\\n") && !val.includes("\\r"),
    "Newline characters are not allowed",
  );
