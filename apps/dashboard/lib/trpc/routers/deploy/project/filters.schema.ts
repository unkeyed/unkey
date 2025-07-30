import { z } from "zod";

export const projectsFilterOperatorEnum = z.enum(["contains"]);

export const projectsListFilterFieldNames = ["name"] as const;

const filterItemSchema = z.object({
  operator: projectsFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

const filterFieldsSchema = projectsListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<string, typeof baseFilterArraySchema>
);

const baseProjectsSchema = z.object(filterFieldsSchema);

export const projectsQueryPayload = baseProjectsSchema.extend({
  cursor: z.number().int().optional(),
});

export type ProjectsQueryPayload = z.infer<typeof projectsQueryPayload>;
