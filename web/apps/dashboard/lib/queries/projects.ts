"use client";
import { trpcClient } from "@/lib/collections/client";
import { useFlag } from "@/lib/flags/provider";
import { DEPLOY_DATA_STALE_MS } from "@/lib/queries/stale-time";
import { queryKeys } from "@/lib/query-keys";
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
  refetchInterval,
}: { select?: (projects: Project[]) => T; enabled?: boolean; refetchInterval?: number } = {}) {
  return useQuery({
    queryKey: queryKeys.projects.all,
    staleTime: DEPLOY_DATA_STALE_MS,
    retry: 3,
    enabled,
    select,
    refetchInterval,
    queryFn: ({ signal }) => loadProjects(signal),
  });
}

// The default project lives outside the public API, so it comes from tRPC
export async function loadProjects(signal?: AbortSignal): Promise<Project[]> {
  const [pages, defaultProject] = await Promise.all([
    getUnkeyClient().projects.listProjects({ limit: 100 }, { signal }),
    trpcClient.deploy.project.getDefault.query(undefined, { signal }),
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

// Every namespace lives in the workspace default project
export function useDefaultProjectId({ enabled }: { enabled: boolean }) {
  return useProjects({ enabled, select: (projects) => projects.find((p) => p.isDefault)?.id });
}

export function useVisibleProjects({ refetchInterval }: { refetchInterval?: number } = {}) {
  const projectsNav = useFlag("projectsNav");
  return useProjects({
    select: (projects) => visibleProjects(projects, projectsNav),
    refetchInterval,
  });
}

export function visibleProjects(projects: readonly Project[], includeDefault: boolean): Project[] {
  return projects
    .filter((p) => includeDefault || !p.isDefault)
    .toSorted((a, b) => b.createdAt - a.createdAt);
}

// Seeds the list with the new project, so its pages find it on arrival even if the refetch fails
export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateProjectRequestSchema) => {
      const { data } = await toastRequest(getUnkeyClient().projects.createProject(input), {
        loading: "Creating project...",
        success: "Project created successfully",
        failure: "Failed to Create Project",
      });
      return data.id;
    },
    onSuccess: (id, { name, slug }) => {
      queryClient.setQueryData<Project[]>(queryKeys.projects.all, (projects) =>
        projects?.concat({ id, name, slug, isDefault: false, createdAt: Date.now() }),
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}

export function useRenameProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ projectId, name }: { projectId: string; name: string }) => {
      await toastRequest(getUnkeyClient().projects.updateProject({ project: projectId, name }), {
        loading: "Updating project...",
        success: "Project updated successfully",
        failure: "Failed to Update Project",
      });
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
      await toastRequest(getUnkeyClient().projects.deleteProject({ project: projectId }), {
        loading: "Deleting project...",
        success: "Project deleted successfully",
        failure: "Failed to Delete Project",
      });
    },
    // A refetch here could drop the project while its page is still mounted, so
    // ProjectGuard would 404 before the redirect lands. The list refetches on mount.
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all, refetchType: "none" }),
  });
}

function toastRequest<T>(
  request: Promise<T>,
  labels: { loading: string; success: string; failure: string },
): Promise<T> {
  toast.promise(request, {
    loading: labels.loading,
    success: labels.success,
    error: (err) => {
      console.error(labels.failure, err);
      return getErrorToast(err, labels.failure);
    },
  });
  return request;
}
