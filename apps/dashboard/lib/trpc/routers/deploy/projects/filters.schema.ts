import { z } from "zod";

// Filter operators
export const projectsFilterOperatorEnum = z.enum(["contains"]);

// Filter item schema
const filterItemSchema = z.object({
  operator: projectsFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

// Query payload schema
export const projectsInputSchema = z.object({
  name: baseFilterArraySchema,
  slug: baseFilterArraySchema,
  branch: baseFilterArraySchema,
  cursor: z.number().int().optional(),
});

// Project response schema
export const projectSchema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  gitRepositoryUrl: z.string().nullable(),
  branch: z.string().nullable(),
  deleteProtection: z.boolean().nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

// Projects list response schema
export const projectsResponseSchema = z.object({
  projects: z.array(projectSchema),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().nullish(),
});

// Exported types
export type ProjectsQueryPayload = z.infer<typeof projectsInputSchema>;
export type Project = z.infer<typeof projectSchema>;
export type ProjectsQueryResponse = z.infer<typeof projectsResponseSchema>;

// Constants
export const PROJECTS_LIMIT = 50;
export const FILTERABLE_FIELDS = ["name", "slug", "branch"] as const;
export type FilterableField = (typeof FILTERABLE_FIELDS)[number];
