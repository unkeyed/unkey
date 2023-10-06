import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { unkeyRoot } from "@/lib/api";
import { and, db, eq, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { Vercel } from "@unkey/vercel";
import { auth, t } from "../trpc";

export const vercelRouter = t.router({
  upsertApiId: t.procedure
    .use(auth)
    .input(
      z.object({
        projectId: z.string(),
        integrationId: z.string(),
        apiId: z.string(),
        environment: z.enum(["production", "preview", "development"]),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          vercelBindings: true,
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (integration.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }

      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });

      const res = await vercel.upsertEnvironmentVariable(
        input.projectId,
        input.environment,
        "UNKEY_API_ID",
        input.apiId,
      );
      const existingBinding = integration.vercelBindings.find(
        (b) =>
          b.projectId === input.projectId &&
          b.environment === input.environment &&
          b.resourceType === "apiId",
      );
      if (existingBinding) {
        await db
          .update(schema.vercelBindings)
          .set({
            resourceId: input.apiId,
            vercelEnvId: res.created.id,
            updatedAt: new Date(),
            lastEditedBy: ctx.user.id,
          })
          .where(eq(schema.vercelBindings.id, existingBinding.id));
      } else {
        await db.insert(schema.vercelBindings).values({
          id: newId("vercelBinding"),
          createdAt: new Date(Date.now()),
          updatedAt: new Date(Date.now()),
          resourceType: "apiId",
          resourceId: input.apiId,
          vercelEnvId: res.created.id,
          lastEditedBy: ctx.user.id,
          environment: input.environment,
          projectId: input.projectId,
          workspaceId: integration.workspace.id,
          integrationId: integration.id,
        });
      }
    }),
  upsertNewRootKey: t.procedure
    .use(auth)
    .input(
      z.object({
        projectId: z.string(),
        integrationId: z.string(),
        environment: z.enum(["production", "preview", "development"]),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          vercelBindings: true,
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (integration.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }

      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });

      const newRootKey = await unkeyRoot._internal.createRootKey({
        name: `Vercel Integration - ${input.environment}`,
        forWorkspaceId: integration.workspace.id,
      });
      if (newRootKey.error) {
        console.error(newRootKey.error.message);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "unable to create new rootKey",
        });
      }
      const res = await vercel.upsertEnvironmentVariable(
        input.projectId,
        input.environment,
        "UNKEY_ROOT_KEY",
        newRootKey.result.key,
        true,
      );
      const existingBinding = integration.vercelBindings.find(
        (b) =>
          b.projectId === input.projectId &&
          b.environment === input.environment &&
          b.resourceType === "rootKey",
      );
      if (existingBinding) {
        await db
          .update(schema.vercelBindings)
          .set({
            resourceId: newRootKey.result.keyId,
            vercelEnvId: res.created.id,
            updatedAt: new Date(),
            lastEditedBy: ctx.user.id,
          })
          .where(eq(schema.vercelBindings.id, existingBinding.id));
      } else {
        console.log("inserting new root key binding");
        await db.insert(schema.vercelBindings).values({
          id: newId("vercelBinding"),
          createdAt: new Date(Date.now()),
          updatedAt: new Date(Date.now()),
          resourceType: "rootKey",
          resourceId: newRootKey.result.keyId,
          vercelEnvId: res.created.id,
          lastEditedBy: ctx.user.id,
          environment: input.environment,
          projectId: input.projectId,
          workspaceId: integration.workspace.id,
          integrationId: integration.id,
        });
      }
    }),
  unbind: t.procedure
    .use(auth)
    .input(
      z.object({
        bindingId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const binding = await db.query.vercelBindings.findFirst({
        where: eq(schema.vercelBindings.id, input.bindingId),
        with: {
          vercelIntegrations: {
            with: {
              workspace: true,
            },
          },
        },
      });
      if (!binding) {
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (binding.vercelIntegrations.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }
      const vercel = new Vercel({
        accessToken: binding.vercelIntegrations.accessToken,
        teamId: binding.vercelIntegrations.vercelTeamId ?? undefined,
      });

      await vercel.removeEnvironmentVariable(binding.projectId, binding.vercelEnvId);
      await db.delete(schema.vercelBindings).where(eq(schema.vercelBindings.id, binding.id));
    }),
  disconnectProject: t.procedure
    .use(auth)
    .input(
      z.object({
        projectId: z.string(),
        integrationId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      console.log("disconnecting project", input.projectId);
      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          vercelBindings: true,
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (integration.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }
      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });

      const bindings = await db.query.vercelBindings.findMany({
        where: and(eq(schema.vercelBindings.projectId, input.projectId)),
      });

      for (const binding of bindings) {
        await vercel.removeEnvironmentVariable(binding.projectId, binding.vercelEnvId);
        await db.delete(schema.vercelBindings).where(eq(schema.vercelBindings.id, binding.id));
      }
    }),
});
