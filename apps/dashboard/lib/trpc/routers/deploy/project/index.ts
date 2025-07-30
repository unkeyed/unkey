import {
  ratelimit,
  requireUser,
  requireWorkspace,
  t,
  withRatelimit,
} from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { db } from "@/lib/db";
import { eq, lt, like, or, and, desc, schema, count } from "@unkey/db";
import {
  projectsInputSchema,
  ProjectsQueryResponse,
  projectsResponseSchema,
} from "./filters.schema";

export const LIMIT = 20;

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
      baseConditions.push(lt(schema.projects.updatedAt, input.cursor));
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

    // Default branch filter
    if (input.branch && input.branch.length > 0) {
      const branchConditions = input.branch.map((filter) => {
        if (filter.operator === "contains") {
          return like(schema.projects.defaultBranch, `%${filter.value}%`);
        }
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Unsupported defaultBranch operator: ${filter.operator}`,
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
      filterConditions.length > 0
        ? [...baseConditions, ...filterConditions]
        : baseConditions;

    try {
      const [totalResult, projectsResult] = await Promise.all([
        db
          .select({ count: count() })
          .from(schema.projects)
          .where(and(...allConditions)),
        db.query.projects.findMany({
          where: and(...allConditions),
          orderBy: [
            desc(schema.projects.updatedAt),
            desc(schema.projects.createdAt),
            desc(schema.projects.id),
          ],
          limit: LIMIT + 1, // Get one extra to check if there are more
          columns: {
            id: true,
            workspaceId: true,
            partitionId: true,
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
      const hasMore = projectsResult.length > LIMIT;
      const projectsWithoutExtra = hasMore
        ? projectsResult.slice(0, LIMIT)
        : projectsResult;

      const response: ProjectsQueryResponse = {
        projects: projectsWithoutExtra.map((p) => ({
          ...p,
          branch: p.defaultBranch,
        })),
        hasMore,
        total: totalResult[0]?.count ?? 0,
        nextCursor:
          projectsWithoutExtra.length > 0
            ? projectsWithoutExtra[projectsWithoutExtra.length - 1].updatedAt
            : undefined,
      };

      return response;
    } catch (error) {
      console.error("Error querying projects:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve projects due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }
  });
