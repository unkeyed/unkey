import { z } from "zod";

export const identityExternalIdSchema = z
  .string()
  .trim()
  .min(1, "External ID must be at least 1 character")
  .max(255, "External ID cannot exceed 255 characters")
  .regex(
    /^[a-zA-Z0-9_.-]+$/,
    "External ID can only contain letters, numbers, underscores, dots, and hyphens",
  );
