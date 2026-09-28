import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { and, db, desc, eq, isNotNull, ne, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import {
  bindingEndpointsSchema,
  bindingHostVariable,
  bindingNameSchema,
  defaultBindingName,
} from "./validation";

const target = z.discriminatedUnion("targetType", [
  z.object({ targetType: z.literal("automatic") }),
  z.object({ targetType: z.literal("environment"), targetEnvironmentId: z.string().min(1) }),
  z.object({ targetType: z.literal("deployment"), targetDeploymentId: z.string().min(1) }),
]);
type Target = z.infer<typeof target>;
const projectInput = z.object({ projectId: z.string().min(1) });
type Endpoints = z.infer<typeof bindingEndpointsSchema>;
const optionalName = z.object({ name: bindingNameSchema.optional() });
const duplicateMessage = "This app is already connected, or the name is already in use.";

export const listAppBindings = workspaceProcedure
  .input(
    projectInput.extend({
      appId: z.string().min(1).optional(),
      environmentId: z.string().min(1).optional(),
      targetAppId: z.string().min(1).optional(),
    }),
  )
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    await requireProject(ctx.workspace.id, input.projectId);
    const rows = await db
      .select({
        id: schema.appBindings.id,
        appId: schema.appBindings.appId,
        environmentId: schema.appBindings.environmentId,
        targetAppId: schema.appBindings.resourceId,
        targetAppName: schema.apps.name,
        targetAppSlug: schema.apps.slug,
        name: schema.appBindings.name,
        targetType: schema.appBindings.selectionMode,
        targetEnvironmentId: schema.appBindings.targetEnvironmentId,
        targetDeploymentId: schema.appBindings.targetDeploymentId,
      })
      .from(schema.appBindings)
      .innerJoin(schema.apps, eq(schema.apps.id, schema.appBindings.resourceId))
      .where(
        and(
          eq(schema.appBindings.workspaceId, ctx.workspace.id),
          eq(schema.appBindings.projectId, input.projectId),
          eq(schema.appBindings.resourceType, "app"),
          ne(schema.appBindings.resourceId, schema.appBindings.appId),
          eq(schema.apps.workspaceId, ctx.workspace.id),
          eq(schema.apps.projectId, input.projectId),
          input.appId ? eq(schema.appBindings.appId, input.appId) : undefined,
          input.environmentId
            ? eq(schema.appBindings.environmentId, input.environmentId)
            : undefined,
          input.targetAppId ? eq(schema.appBindings.resourceId, input.targetAppId) : undefined,
        ),
      )
      .orderBy(schema.appBindings.name)
      .limit(500);
    return rows.map((row) => ({ ...row, ...target.parse(row) }));
  });

export const listAppBindingTargets = workspaceProcedure
  .input(projectInput.extend({ appId: z.string().min(1) }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    await requireApp(ctx.workspace.id, input.projectId, input.appId);
    const [apps, environments, deployments] = await Promise.all([
      db
        .select({ id: schema.apps.id, name: schema.apps.name, slug: schema.apps.slug })
        .from(schema.apps)
        .where(
          and(
            eq(schema.apps.workspaceId, ctx.workspace.id),
            eq(schema.apps.projectId, input.projectId),
            ne(schema.apps.id, input.appId),
          ),
        )
        .orderBy(schema.apps.name)
        .limit(100),
      db
        .select({
          id: schema.environments.id,
          appId: schema.environments.appId,
          slug: schema.environments.slug,
          kind: schema.environments.kind,
        })
        .from(schema.environments)
        .where(
          and(
            eq(schema.environments.workspaceId, ctx.workspace.id),
            eq(schema.environments.projectId, input.projectId),
          ),
        )
        .orderBy(schema.environments.slug)
        .limit(500),
      db
        .select({
          id: schema.deployments.id,
          appId: schema.deployments.appId,
          environmentId: schema.deployments.environmentId,
          status: schema.deployments.status,
          gitBranch: schema.deployments.gitBranch,
          image: schema.deployments.imageResolved,
        })
        .from(schema.deployments)
        .where(
          and(
            eq(schema.deployments.workspaceId, ctx.workspace.id),
            eq(schema.deployments.projectId, input.projectId),
            isNotNull(schema.deployments.firstReadyAt),
          ),
        )
        .orderBy(desc(schema.deployments.createdAt))
        .limit(100),
    ]);
    return {
      apps,
      environments,
      deployments: deployments.filter(
        (item) => item.status === "ready" || item.status === "stopped",
      ),
    };
  });

export const createAppBinding = workspaceProcedure
  .input(projectInput.and(bindingEndpointsSchema).and(optionalName).and(target))
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    const { targetSlug } = await validateEndpoints(ctx.workspace.id, input.projectId, input);
    await validateTarget(ctx.workspace.id, input.projectId, input.targetAppId, input);
    const name = input.name
      ? await requireUnusedName(ctx.workspace.id, input, input.name)
      : await pickDefaultName(ctx.workspace.id, input, targetSlug);
    const id = `bind_${crypto.randomUUID().replaceAll("-", "").slice(0, 20)}`;
    await mutateWithDuplicateHandling(async () =>
      db.transaction(async (tx) => {
        await lockPinnedTarget(tx, ctx.workspace.id, input.projectId, input.targetAppId, input);
        await tx.insert(schema.appBindings).values({
          id,
          workspaceId: ctx.workspace.id,
          projectId: input.projectId,
          appId: input.appId,
          environmentId: input.environmentId,
          resourceType: "app",
          resourceId: input.targetAppId,
          name,
          ...targetColumns(input),
          createdAt: Date.now(),
        });
        await audit(tx, ctx, input.appId, `Created app binding ${name}`);
      }),
    );
    return { id, name };
  });

export const updateAppBinding = workspaceProcedure
  .input(
    projectInput
      .extend({ id: z.string().min(1) })
      .and(optionalName)
      .and(target),
  )
  .use(withRatelimit(ratelimit.update))
  .mutation(async ({ ctx, input }) => {
    const binding = await requireBinding(ctx.workspace.id, input.projectId, input.id);
    await validateTarget(ctx.workspace.id, input.projectId, binding.resourceId, input);
    const name = await requireUnusedName(
      ctx.workspace.id,
      binding,
      input.name ?? binding.name,
      binding.id,
    );
    await mutateWithDuplicateHandling(async () =>
      db.transaction(async (tx) => {
        await lockPinnedTarget(tx, ctx.workspace.id, input.projectId, binding.resourceId, input);
        await tx
          .update(schema.appBindings)
          .set({ name, ...targetColumns(input), updatedAt: Date.now() })
          .where(
            and(
              eq(schema.appBindings.id, input.id),
              eq(schema.appBindings.workspaceId, ctx.workspace.id),
              eq(schema.appBindings.projectId, input.projectId),
            ),
          );
        await audit(tx, ctx, binding.appId, `Updated app binding ${name}`);
      }),
    );
    return { id: input.id, name };
  });

export const deleteAppBinding = workspaceProcedure
  .input(projectInput.extend({ id: z.string().min(1) }))
  .use(withRatelimit(ratelimit.delete))
  .mutation(async ({ ctx, input }) => {
    const binding = await requireBinding(ctx.workspace.id, input.projectId, input.id);
    await db.transaction(async (tx) => {
      await tx
        .delete(schema.appBindings)
        .where(
          and(
            eq(schema.appBindings.id, input.id),
            eq(schema.appBindings.workspaceId, ctx.workspace.id),
            eq(schema.appBindings.projectId, input.projectId),
          ),
        );
      await audit(tx, ctx, binding.appId, `Deleted app binding ${binding.name}`);
    });
    return { id: input.id };
  });

function targetColumns(input: Target) {
  return {
    selectionMode: input.targetType,
    targetEnvironmentId: input.targetType === "environment" ? input.targetEnvironmentId : null,
    targetDeploymentId: input.targetType === "deployment" ? input.targetDeploymentId : null,
  };
}

async function validateEndpoints(workspaceId: string, projectId: string, input: Endpoints) {
  const [, targetApp, callerEnvironment, existing] = await Promise.all([
    requireApp(workspaceId, projectId, input.appId),
    requireApp(workspaceId, projectId, input.targetAppId),
    db.query.environments.findFirst({
      columns: { id: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.id, input.environmentId),
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, input.appId),
        ),
    }),
    db.query.appBindings.findFirst({
      columns: { id: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, input.appId),
          eq(t.environmentId, input.environmentId),
          eq(t.resourceType, "app"),
          eq(t.resourceId, input.targetAppId),
        ),
    }),
  ]);
  if (!callerEnvironment) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Caller environment not found." });
  }
  if (existing) {
    throw new TRPCError({ code: "CONFLICT", message: "This app is already connected." });
  }
  return { targetSlug: targetApp.slug };
}

async function validateTarget(
  workspaceId: string,
  projectId: string,
  targetAppId: string,
  input: Target,
) {
  if (input.targetType === "environment") {
    const env = await db.query.environments.findFirst({
      columns: { id: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.id, input.targetEnvironmentId),
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, targetAppId),
        ),
    });
    if (!env) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Select an environment from the target app.",
      });
    }
  }
  if (input.targetType === "deployment") {
    const deployment = await db.query.deployments.findFirst({
      columns: { id: true, status: true, firstReadyAt: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.id, input.targetDeploymentId),
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, targetAppId),
        ),
    });
    if (
      !deployment ||
      !deployment.firstReadyAt ||
      !["ready", "stopped"].includes(deployment.status)
    ) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Select an approved deployment that reached ready.",
      });
    }
  }
}

async function takenNames(
  workspaceId: string,
  scope: { appId: string; environmentId: string },
  excludeId?: string,
) {
  const [bindings, variables] = await Promise.all([
    db.query.appBindings.findMany({
      columns: { id: true, name: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.workspaceId, workspaceId),
          eq(t.appId, scope.appId),
          eq(t.environmentId, scope.environmentId),
        ),
    }),
    db.query.appEnvironmentVariables.findMany({
      columns: { key: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.workspaceId, workspaceId),
          eq(t.appId, scope.appId),
          eq(t.environmentId, scope.environmentId),
        ),
    }),
  ]);
  const names = new Set(bindings.filter((b) => b.id !== excludeId).map((b) => b.name));
  const keys = new Set(variables.map((v) => v.key));
  return (name: string) => names.has(name) || keys.has(bindingHostVariable(name));
}

async function pickDefaultName(workspaceId: string, scope: Endpoints, targetSlug: string) {
  const name = defaultBindingName(targetSlug, await takenNames(workspaceId, scope));
  if (!name) {
    throw new TRPCError({ code: "CONFLICT", message: "Choose a name for this binding." });
  }
  return name;
}

async function requireUnusedName(
  workspaceId: string,
  scope: { appId: string; environmentId: string },
  name: string,
  excludeId?: string,
) {
  const isTaken = await takenNames(workspaceId, scope, excludeId);
  if (isTaken(name)) {
    throw new TRPCError({
      code: "CONFLICT",
      message: `${name} or ${bindingHostVariable(name)} is already used in this environment.`,
    });
  }
  return name;
}

async function requireProject(workspaceId: string, projectId: string) {
  const row = await db.query.projects.findFirst({
    columns: { id: true },
    where: (t, { and, eq }) => and(eq(t.id, projectId), eq(t.workspaceId, workspaceId)),
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Project not found." });
  }
}
async function requireApp(workspaceId: string, projectId: string, appId: string) {
  const row = await db.query.apps.findFirst({
    columns: { id: true, slug: true },
    where: (t, { and, eq }) =>
      and(eq(t.id, appId), eq(t.projectId, projectId), eq(t.workspaceId, workspaceId)),
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "App not found." });
  }
  return row;
}
async function requireBinding(workspaceId: string, projectId: string, id: string) {
  const row = await db.query.appBindings.findFirst({
    columns: { id: true, appId: true, environmentId: true, resourceId: true, name: true },
    where: (t, { and, eq }) =>
      and(
        eq(t.id, id),
        eq(t.workspaceId, workspaceId),
        eq(t.projectId, projectId),
        eq(t.resourceType, "app"),
      ),
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Binding not found." });
  }
  return row;
}
async function audit(
  tx: Parameters<typeof insertAuditLogs>[0],
  ctx: { workspace: { id: string }; user: { id: string }; audit: UnkeyAuditLog["context"] },
  appId: string,
  description: string,
) {
  await insertAuditLogs(tx, {
    workspaceId: ctx.workspace.id,
    actor: { type: "user", id: ctx.user.id },
    event: "app.update",
    description,
    resources: [{ type: "app", id: appId }],
    context: ctx.audit,
  });
}
async function lockPinnedTarget(
  tx: Parameters<typeof insertAuditLogs>[0],
  workspaceId: string,
  projectId: string,
  targetAppId: string,
  input: Target,
) {
  if (input.targetType !== "deployment") {
    return;
  }
  const [deployment] = await tx
    .select({ status: schema.deployments.status, desiredState: schema.deployments.desiredState })
    .from(schema.deployments)
    .where(
      and(
        eq(schema.deployments.id, input.targetDeploymentId),
        eq(schema.deployments.workspaceId, workspaceId),
        eq(schema.deployments.projectId, projectId),
        eq(schema.deployments.appId, targetAppId),
      ),
    )
    .for("update");
  if (deployment?.status !== "ready" || deployment.desiredState !== "running") {
    throw new TRPCError({
      code: "CONFLICT",
      message: "The target deployment is no longer running. Select a ready deployment.",
    });
  }
}

async function mutateWithDuplicateHandling(fn: () => Promise<unknown>) {
  try {
    await fn();
  } catch (error) {
    if (
      error instanceof Error &&
      (("code" in error && error.code === "ER_DUP_ENTRY") ||
        (error.cause instanceof Error &&
          "code" in error.cause &&
          error.cause.code === "ER_DUP_ENTRY"))
    ) {
      throw new TRPCError({ code: "CONFLICT", message: duplicateMessage });
    }
    throw error;
  }
}
