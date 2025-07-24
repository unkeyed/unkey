// trpc/routers/versions/getOpenApiDiff.ts
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const getOpenApiDiff = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      oldVersionId: z.string(),
      newVersionId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    try {
      // verify both versions exist and belong to this workspace
      const [oldVersion, newVersion] = await Promise.all([
        db.query.versions.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.id, input.oldVersionId), eq(table.workspaceId, ctx.workspace.id)),
          columns: {
            id: true,
            openapiSpec: true,
            gitCommitSha: true,
            gitBranch: true,
            // TODO: add this column
            //gitCommitMessage: true,
          },
          with: {
            project: {
              columns: {
                id: true,
                name: true,
                slug: true,
              },
            },
            branch: {
              columns: {
                id: true,
                name: true,
              },
            },
          },
        }),
        db.query.versions.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.id, input.newVersionId), eq(table.workspaceId, ctx.workspace.id)),
          columns: {
            id: true,
            openapiSpec: true,
            gitCommitSha: true,
            gitBranch: true,
            //gitCommitMessage: true,
          },
          with: {
            project: {
              columns: {
                id: true,
                name: true,
                slug: true,
              },
            },
            branch: {
              columns: {
                id: true,
                name: true,
              },
            },
          },
        }),
      ]);

      if (!oldVersion || !newVersion) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "One or both versions not found",
        });
      }

      if (!oldVersion.openapiSpec || !newVersion.openapiSpec) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "OpenAPI spec not available for one or both versions",
        });
      }

      // Call control plane API
      // TODO: put this in env var
      let diffData;
      
      try {
        const response = await fetch("http://localhost:7091/ctrl.v1.OpenApiService/GetOpenApiDiff", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            old_version_id: input.oldVersionId,
            new_version_id: input.newVersionId,
          }),
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        diffData = await response.json();
      } catch (error) {
        // Fallback to mock data if control plane is not available
        console.warn("Control plane not available, using mock diff data:", error);
        diffData = {
          changes: [
            {
              id: "change_1",
              text: "Added new endpoint /api/v1/users",
              level: 1,
              operation: "POST",
              path: "/api/v1/users",
              source: "paths",
              section: "paths",
              comment: "New user creation endpoint"
            },
            {
              id: "change_2", 
              text: "Modified response schema for /api/v1/users/{id}",
              level: 2,
              operation: "GET",
              path: "/api/v1/users/{id}",
              source: "responses",
              section: "responses",
              comment: "Added new fields to user response"
            },
            {
              id: "change_3",
              text: "Removed deprecated endpoint /api/v1/legacy",
              level: 3,
              operation: "DELETE",
              path: "/api/v1/legacy",
              source: "paths",
              section: "paths",
              comment: "Breaking change: endpoint removed"
            }
          ]
        };
      }

      // Ensure the diff data has the expected structure
      const normalizedDiff = {
        changes: Array.isArray(diffData?.changes) ? diffData.changes : [],
        ...diffData
      };

      // Return the diff along with version context for the UI
      return {
        diff: normalizedDiff,
        context: {
          oldVersion: {
            id: oldVersion.id,
            gitCommitSha: oldVersion.gitCommitSha,
            gitBranch: oldVersion.gitBranch,
            //gitCommitMessage: oldVersion.gitCommitMessage,
            project: oldVersion.project,
            branch: oldVersion.branch,
          },
          newVersion: {
            id: newVersion.id,
            gitCommitSha: newVersion.gitCommitSha,
            gitBranch: newVersion.gitBranch,
            //gitCommitMessage: newVersion.gitCommitMessage,
            project: newVersion.project,
            branch: newVersion.branch,
          },
          isSameBranch: oldVersion.gitBranch === newVersion.gitBranch,
          isSameProject: oldVersion.project?.id === newVersion.project?.id,
        },
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Failed to get OpenAPI diff:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to get OpenAPI diff",
      });
    }
  });
