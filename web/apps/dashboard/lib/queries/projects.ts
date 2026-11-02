"use client";
import { useFlag } from "@/lib/flags/provider";
import { DEPLOY_DATA_STALE_MS } from "@/lib/queries/stale-time";
import { queryKeys } from "@/lib/query-keys";
import { trpc } from "@/lib/trpc/client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { z } from "zod";

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

export type CreateProjectRequestSchema = z.infer<typeof createProjectRequestSchema>;

export type Project = {
  id: string;
  name: string;
  slug: string;
  isDefault: boolean;
  createdAt: number;
};

export function projectDisplayName(
  project: Pick<Project, "name" | "isDefault">,
  workspaceName: string,
): string {
  return project.isDefault ? workspaceName : project.name;
}

export function useProjects<T = Project[]>({
  select,
  enabled = true,
}: { select?: (projects: Project[]) => T; enabled?: boolean } = {}) {
  const trpcUtils = trpc.useUtils();
  return useQuery({
    queryKey: queryKeys.projects.all,
    staleTime: DEPLOY_DATA_STALE_MS,
    retry: 3,
    enabled,
    select,
    queryFn: ({ signal }) =>
      loadProjects(
        () => trpcUtils.client.deploy.project.getDefault.query(undefined, { signal }),
        signal,
      ),
  });
}

// The default project lives outside the public API, so it comes from tRPC
export async function loadProjects(
  getDefault: () => Promise<Omit<Project, "isDefault"> | null>,
  signal?: AbortSignal,
): Promise<Project[]> {
  const [pages, defaultProject] = await Promise.all([
    getUnkeyClient().projects.listProjects({ limit: 100 }, { signal }),
    getDefault(),
  ]);
  const listed = (await Array.fromAsync(pages)).flatMap((page) =>
    page.result.data.map(
      (p): Project => ({
        id: p.id,
        name: p.name,
        slug: p.slug,
        isDefault: false,
        createdAt: p.createdAt,
      }),
    ),
  );
  return defaultProject ? [{ ...defaultProject, isDefault: true }, ...listed] : listed;
}

// Null once the projects have loaded without this one
export function useProject(projectId: string | undefined) {
  return useProjects({
    enabled: projectId !== undefined,
    select: (projects) => projects.find((p) => p.id === projectId) ?? null,
  });
}

export function useVisibleProjects() {
  const projectsNav = useFlag("projectsNav");
  return useProjects({ select: (projects) => visibleProjects(projects, projectsNav) });
}

export function visibleProjects(projects: readonly Project[], includeDefault: boolean): Project[] {
  return projects
    .filter((p) => includeDefault || !p.isDefault)
    .toSorted((a, b) => b.createdAt - a.createdAt);
}

// Resolves to the new project's id once the list holds it, so its pages find it on arrival
export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateProjectRequestSchema) => {
      const request = getUnkeyClient().projects.createProject(
        createProjectRequestSchema.parse(input),
      );
      toast.promise(request, {
        loading: "Creating project...",
        success: "Project created successfully",
        error: (err) => {
          console.error("Failed to create project", err);
          return getErrorToast(err, "Failed to Create Project");
        },
      });
      return (await request).data.id;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.projects.all }),
  });
}

export function useRenameProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ projectId, name }: { projectId: string; name: string }) => {
      const request = getUnkeyClient().projects.updateProject({
        project: projectId,
        ...createProjectRequestSchema.pick({ name: true }).parse({ name }),
      });
      toast.promise(request, {
        loading: "Updating project...",
        success: "Project updated successfully",
        error: (err) => {
          console.error("Failed to update project", err);
          return getErrorToast(err, "Failed to Update Project");
        },
      });
      await request;
    },
    onMutate: async ({ projectId, name }) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.projects.all });
      const previous = queryClient.getQueryData<Project[]>(queryKeys.projects.all);
      queryClient.setQueryData<Project[]>(queryKeys.projects.all, (projects) =>
        projects?.map((p) => (p.id === projectId ? { ...p, name } : p)),
      );
      return { previous };
    },
    onError: (_error, _input, context) => {
      queryClient.setQueryData(queryKeys.projects.all, context?.previous);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: queryKeys.projects.all }),
  });
}

// The API queues the delete, so the list can still hold the project until the workflow finishes
export function useDeleteProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (projectId: string) => {
      const request = getUnkeyClient().projects.deleteProject({ project: projectId });
      toast.promise(request, {
        loading: "Deleting project...",
        success: "Project deleted successfully",
        error: (err) => {
          console.error("Failed to delete project", err);
          return getErrorToast(err, "Failed to Delete Project");
        },
      });
      await request;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}
