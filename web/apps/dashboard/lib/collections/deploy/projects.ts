import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  // Apps inside the project, newest first, for the card's app stack.
  appCount: z.number().int(),
  apps: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      source: z.enum(["github", "docker", "code"]),
      repository: z.string().nullable(),
    }),
  ),
  repositoryFullName: z.string().nullable(),
  latestDeploymentId: z.string().nullable(),
  currentDeploymentId: z.string().nullable(),
  isRolledBack: z.boolean(),
  // Flattened deployment fields for UI
  commitTitle: z.string().nullable(),
  commitSha: z.string().nullable(),
  forkRepositoryFullName: z.string().nullable(),
  prNumber: z.number().int().nullable(),
  branch: z.string(),
  author: z.string().nullable(),
  authorAvatar: z.string().nullable(),
  commitTimestamp: z.number().int().nullable(),
  // Domain field
  domain: z.string().nullable(),
});

export const createProjectRequestSchema = z.object({
  name: z.string().trim().min(1, "Project name is required").max(256, "Project name too long"),
  slug: z
    .string()
    .trim()
    .min(3, "Project slug must be at least 3 characters")
    .max(256, "Project slug too long")
    .regex(
      /^[a-zA-Z0-9_-]+$/,
      "Project slug must contain only letters, numbers, hyphens, and underscores",
    ),
});

export type Project = z.infer<typeof schema>;
export type CreateProjectRequestSchema = z.infer<typeof createProjectRequestSchema>;

export const projects = createCollection<Project, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["projects"],
    retry: 3,
    queryFn: async () => {
      return await trpcClient.deploy.project.list.query();
    },
    getKey: (item) => item.id,
    onDelete: async ({ transaction }) => {
      const mutation = transaction.mutations[0];
      const projectId = mutation.original.id;

      const deleteMutation = getUnkeyClient().projects.deleteProject({ project: projectId });

      toast.promise(deleteMutation, {
        loading: "Deleting project...",
        success: "Project deleted successfully",
        error: (err) => {
          console.error("Failed to delete project", err);
          return getErrorToast(err, "Failed to Delete Project");
        },
      });

      await deleteMutation;
      // Automatically refetches query after delete
    },
    onInsert: async ({ transaction }) => {
      const { changes } = transaction.mutations[0];

      const createInput = createProjectRequestSchema.parse({
        name: changes.name,
        slug: changes.slug,
      });
      const mutation = getUnkeyClient().projects.createProject(createInput);

      toast.promise(mutation, {
        loading: "Creating project...",
        success: "Project created successfully",
        error: (err) => {
          console.error("Failed to create project", err);
          return getErrorToast(err, "Failed to Create Project");
        },
      });
      const result = await mutation;
      transaction.metadata = {
        projectId: result.data.id,
      };
    },
    onUpdate: async ({ transaction }) => {
      const { original, changes } = transaction.mutations[0];

      const updateInput = {
        project: original.id,
        ...createProjectRequestSchema.pick({ name: true }).parse({
          name: changes.name ?? original.name,
        }),
      };
      const mutation = getUnkeyClient().projects.updateProject(updateInput);

      toast.promise(mutation, {
        loading: "Updating project...",
        success: "Project updated successfully",
        error: (err) => {
          console.error("Failed to update project", err);
          return getErrorToast(err, "Failed to Update Project");
        },
      });

      await mutation;
    },
  }),
);
