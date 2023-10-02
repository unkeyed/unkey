"use server";

import { serverAction } from "@/lib/actions";
import { VercelBinding, and, db, eq, schema } from "@/lib/db";
import { z } from "zod";

export const updateBindings = serverAction({
  input: z.object({
    integrationId: z.string(),
    projectId: z.string(),
    development: z.string(),
    preview: z.string(),
    production: z.string(),
  }),
  handler: async ({ input, ctx }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: eq(schema.workspaces.tenantId, ctx.tenantId),
      with: {
        vercelIntegrations: {
          where: eq(schema.vercelIntegrations.id, input.integrationId),
          with: {
            vercelBindings: true,
          },
        },
      },
    });
    if (!workspace) {
      throw new Error("workspace not found");
    }
    const integration = workspace.vercelIntegrations.at(0);
    if (!integration) {
      throw new Error("integration not found");
    }
    await updateBinding({
      projectId: input.projectId,
      apiId: input.development,
      workspaceId: workspace.id,
      userId: ctx.userId,
      existingBindings: integration.vercelBindings,
      env: "development",
      integrationId: integration.id,
    });
    await updateBinding({
      projectId: input.projectId,
      apiId: input.preview,
      workspaceId: workspace.id,
      userId: ctx.userId,
      existingBindings: integration.vercelBindings,
      env: "preview",
      integrationId: integration.id,
    });
    await updateBinding({
      projectId: input.projectId,
      apiId: input.production,
      workspaceId: workspace.id,
      userId: ctx.userId,
      existingBindings: integration.vercelBindings,
      env: "production",
      integrationId: integration.id,
    });
  },
});

async function updateBinding(req: {
  projectId: string;
  apiId: string; // can be "", meaning we need to unbind
  workspaceId: string;
  userId: string;
  existingBindings: VercelBinding[];
  env: VercelBinding["environment"];
  integrationId: string;
}): Promise<void> {
  if (!req.apiId) {
    // unbind
    console.log("unbinding", req.projectId, req.env);
    await db
      .delete(schema.vercelBindings)
      .where(
        and(
          eq(schema.vercelBindings.projectId, req.projectId),
          eq(schema.vercelBindings.environment, req.env),
        ),
      )
      .execute();
    return;
  }

  const existingBinding = req.existingBindings.find((binding) => binding.environment === req.env);
  if (existingBinding) {
    console.log("updating", req.projectId, req.env);
    await db
      .update(schema.vercelBindings)
      .set({
        apiId: req.apiId,
        updatedAt: new Date(),
        lastEditedBy: req.userId,
      })
      .where(
        and(
          eq(schema.vercelBindings.projectId, req.projectId),
          eq(schema.vercelBindings.environment, req.env),
        ),
      )
      .execute();
  } else {
    console.log("creating", req.projectId, req.env);
    await db
      .insert(schema.vercelBindings)
      .values({
        workspaceId: req.workspaceId,
        integrationId: req.integrationId,
        apiId: req.apiId,
        environment: req.env,
        createdAt: new Date(),
        updatedAt: new Date(),
        lastEditedBy: req.userId,
        rootKeyId: "",
        projectId: req.projectId,
      })
      .execute();
  }
}
