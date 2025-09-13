import { z } from "zod";

export const createProjectSchema = z.object({
  name: z.string().trim().min(1, "Project name is required").max(256, "Project name too long"),
  slug: z
    .string()
    .trim()
    .min(1, "Project slug is required")
    .max(256, "Project slug too long")
    .regex(
      /^[a-z0-9-]+$/,
      "Project slug must contain only lowercase letters, numbers, and hyphens",
    ),
  gitRepositoryUrl: z.string().trim().url("Must be a valid URL").nullable().or(z.literal("")),
});
