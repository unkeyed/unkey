import { and, count, db, desc, eq, like, lt, or, schema } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { PROJECTS_LIMIT, projectsInputSchema, projectsResponseSchema } from "./filters.schema";

export const queryProjects = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(projectsInputSchema)
  .output(projectsResponseSchema)
  .query(async ({ ctx, input }) => {
    // Build base conditions
    const baseConditions = [eq(schema.projects.workspaceId, ctx.workspace.id)];

    // Add cursor condition for pagination
    if (input.cursor && typeof input.cursor === "number") {
      baseConditions.push(lt(schema.projects.createdAt, input.cursor));
    }

    // Build filter conditions
    const filterConditions = [];

    // Name filter
    if (input.name && input.name.length > 0) {
      const nameConditions = input.name.map((filter) => {
        if (filter.operator === "contains") {
          return like(schema.projects.name, `%${filter.value}%`);
        }
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Unsupported name operator: ${filter.operator}`,
        });
      });

      if (nameConditions.length === 1) {
        filterConditions.push(nameConditions[0]);
      } else {
        filterConditions.push(or(...nameConditions));
      }
    }

    // Slug filter
    if (input.slug && input.slug.length > 0) {
      const slugConditions = input.slug.map((filter) => {
        if (filter.operator === "contains") {
          return like(schema.projects.slug, `%${filter.value}%`);
        }
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Unsupported slug operator: ${filter.operator}`,
        });
      });

      if (slugConditions.length === 1) {
        filterConditions.push(slugConditions[0]);
      } else {
        filterConditions.push(or(...slugConditions));
      }
    }

    // Branch filter (searches defaultBranch)
    if (input.branch && input.branch.length > 0) {
      const branchConditions = input.branch.map((filter) => {
        if (filter.operator === "contains") {
          return like(schema.projects.defaultBranch, `%${filter.value}%`);
        }
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Unsupported branch operator: ${filter.operator}`,
        });
      });

      if (branchConditions.length === 1) {
        filterConditions.push(branchConditions[0]);
      } else {
        filterConditions.push(or(...branchConditions));
      }
    }

    // Combine all conditions
    const allConditions =
      filterConditions.length > 0 ? [...baseConditions, ...filterConditions] : baseConditions;

    try {
      const [totalResult, projectsResult] = await Promise.all([
        db
          .select({ count: count() })
          .from(schema.projects)
          .where(and(...allConditions)),
        db.query.projects.findMany({
          where: and(...allConditions),
          orderBy: [desc(schema.projects.createdAt)],
          limit: PROJECTS_LIMIT + 1, // Get one extra to check if there are more
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

      // Fetch hostnames for all projects in a separate query
      const hostnamesResult =
        projectIds.length > 0
          ? await db.query.hostnames.findMany({
              where: and(
                eq(schema.hostnames.workspaceId, ctx.workspace.id),
                or(...projectIds.map((id) => eq(schema.hostnames.projectId, id))),
              ),
              columns: {
                id: true,
                projectId: true,
                hostname: true,
                isCustomDomain: true,
                verificationStatus: true,
                subdomainConfig: true,
              },
              orderBy: [desc(schema.hostnames.createdAt)],
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
            hostname: hostname.hostname,
          });
          return acc;
        },
        {} as Record<
          string,
          Array<{
            id: string;
            hostname: string;
          }>
        >,
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

      const response = {
        projects,
        hasMore,
        total: totalResult[0]?.count ?? 0,
        nextCursor: projects.length > 0 ? projects[projects.length - 1].createdAt : null,
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
