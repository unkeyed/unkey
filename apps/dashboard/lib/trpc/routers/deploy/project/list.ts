import { projectsQueryPayload as projectsInputSchema } from "@/app/(app)/projects/_components/list/projects-list.schema";
import { and, count, db, desc, eq, exists, inArray, like, lt, or, schema } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const HostnameResponse = z.object({
  id: z.string(),
  hostname: z.string(),
});

const ProjectResponse = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  gitRepositoryUrl: z.string().nullable(),
  branch: z.string().nullable(),
  deleteProtection: z.boolean().nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
  hostnames: z.array(HostnameResponse),
});

const projectsOutputSchema = z.object({
  projects: z.array(ProjectResponse),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().nullish(),
});

type ProjectsOutputSchema = z.infer<typeof projectsOutputSchema>;
export type Project = z.infer<typeof ProjectResponse>;

export const PROJECTS_LIMIT = 10;

export const queryProjects = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(projectsInputSchema)
  .output(projectsOutputSchema)
  .query(async ({ ctx, input }) => {
    // Build base conditions
    const baseConditions = [eq(schema.projects.workspaceId, ctx.workspace.id)];

    // Add cursor condition for pagination
    if (input.cursor && typeof input.cursor === "number") {
      baseConditions.push(lt(schema.projects.updatedAt, input.cursor));
    }

    const filterConditions = [];

    // Single query field that searches across name, branch, and hostnames
    if (input.query && input.query.length > 0) {
      const searchConditions = [];

      // Process each query filter
      for (const filter of input.query) {
        if (filter.operator !== "contains") {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: `Unsupported query operator: ${filter.operator}`,
          });
        }

        const searchValue = `%${filter.value}%`;
        const queryConditions = [];

        // Search in project name
        queryConditions.push(like(schema.projects.name, searchValue));

        // Search in project branch (defaultBranch)
        queryConditions.push(like(schema.projects.defaultBranch, searchValue));

        // Search in hostnames
        queryConditions.push(
          exists(
            db
              .select({ projectId: schema.domains.projectId })
              .from(schema.domains)
              .where(
                and(
                  eq(schema.domains.workspaceId, ctx.workspace.id),
                  eq(schema.domains.projectId, schema.projects.id),
                  like(schema.domains.domain, searchValue),
                ),
              ),
          ),
        );

        // Combine all search conditions with OR for this specific query value
        if (queryConditions.length > 0) {
          searchConditions.push(or(...queryConditions));
        }
      }

      if (searchConditions.length > 0) {
        filterConditions.push(or(...searchConditions));
      }
    }

    // Combine all conditions
    const allConditions = [...baseConditions, ...filterConditions];

    try {
      const [totalResult, projectsResult] = await Promise.all([
        db
          .select({ count: count() })
          .from(schema.projects)
          .where(and(...allConditions)),
        db.query.projects.findMany({
          where: and(...allConditions),
          orderBy: [desc(schema.projects.updatedAt)],
          limit: PROJECTS_LIMIT + 1,
          columns: {
            id: true,
            name: true,
            slug: true,
            gitRepositoryUrl: true,
            defaultBranch: true,
            deleteProtection: true,
            createdAt: true,
            updatedAt: true,
          },
        }),
      ]);

      // Check if we have more results
      const hasMore = projectsResult.length > PROJECTS_LIMIT;
      const projectsWithoutExtra = hasMore
        ? projectsResult.slice(0, PROJECTS_LIMIT)
        : projectsResult;

      // Get project IDs for hostname lookup
      const projectIds = projectsWithoutExtra.map((p) => p.id);

      // Fetch hostnames for all projects - only .unkey.app domains
      const hostnamesResult =
        projectIds.length > 0
          ? await db.query.domains.findMany({
              where: and(
                eq(schema.domains.workspaceId, ctx.workspace.id),
                inArray(schema.domains.projectId, projectIds),
                like(schema.domains.domain, "%.unkey.app"),
              ),
              columns: {
                id: true,
                projectId: true,
                domain: true,
              },
              orderBy: [desc(schema.domains.createdAt)],
            })
          : [];

      // Group hostnames by projectId
      const hostnamesByProject = hostnamesResult.reduce(
        (acc, hostname) => {
          if (!acc[hostname.projectId]) {
            acc[hostname.projectId] = [];
          }
          acc[hostname.projectId].push({
            id: hostname.id,
            hostname: hostname.domain,
          });
          return acc;
        },
        {} as Record<string, Array<{ id: string; hostname: string }>>,
      );

      const projects = projectsWithoutExtra.map((project) => ({
        id: project.id,
        name: project.name,
        slug: project.slug,
        gitRepositoryUrl: project.gitRepositoryUrl,
        branch: project.defaultBranch,
        deleteProtection: project.deleteProtection,
        createdAt: project.createdAt,
        updatedAt: project.updatedAt,
        hostnames: hostnamesByProject[project.id] || [],
      }));

      const response: ProjectsOutputSchema = {
        projects,
        hasMore,
        total: totalResult[0]?.count ?? 0,
        nextCursor: projects.length > 0 ? projects[projects.length - 1].updatedAt : null,
      };

      return response;
    } catch (error) {
      console.error("Error querying projects:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve projects due to an error. If this issue persists, please contact support.",
      });
    }
  });
