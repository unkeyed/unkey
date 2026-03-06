import { insertAuditLogs } from "@/lib/audit";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";

import { and, db, eq } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { z } from "zod";

export const createDeploy = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      projectId: z.string().min(1, "Project ID is required"),
      environmentSlug: z.string().min(1, "Environment slug is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const { CTRL_URL, CTRL_API_KEY } = env();
    if (!CTRL_URL || !CTRL_API_KEY) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "ctrl service is not configured",
      });
    }

    const ctrl = createClient(
      DeployService,
      createConnectTransport({
        baseUrl: CTRL_URL,
        interceptors: [
          (next) => (req) => {
            req.header.set("Authorization", `Bearer ${CTRL_API_KEY}`);
            return next(req);
          },
        ],
      }),
    );

    try {
      const project = await db.query.projects.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
        columns: { id: true, name: true },
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found or access denied",
        });
      }

      // Look up the environment to find the app
      const environment = await db.query.environments.findFirst({
        where: and(
          eq(environments.projectId, input.projectId),
          eq(environments.slug, input.environmentSlug),
          eq(environments.workspaceId, ctx.workspace.id),
        ),
        columns: { id: true, appId: true },
      });

      if (!environment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Environment '${input.environmentSlug}' not found`,
        });
      }

      const result = await ctrl
        .createDeployment({
          projectId: input.projectId,
          appId: environment.appId,
          environmentSlug: input.environmentSlug,
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: err,
          });
        });

      await insertAuditLogs(db, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "deployment.create",
        description: `Triggered initial deployment for ${project.name}`,
        resources: [
          {
            type: "deployment",
            id: result.deploymentId,
            name: project.name,
          },
        ],
        context: ctx.audit,
      });

      return { deploymentId: result.deploymentId };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Create deployment request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to communicate with control service",
      });
    }
  });
