import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";
import { getAlertInput } from "./schemas";
import { alertSelection } from "./selection";

export const getAlert = workspaceProcedure.input(getAlertInput).query(async ({ ctx, input }) => {
  const [alert] = await db
    .select(alertSelection)
    .from(schema.alertEvents)
    .innerJoin(schema.apps, eq(schema.apps.id, schema.alertEvents.appId))
    .innerJoin(schema.environments, eq(schema.environments.id, schema.alertEvents.environmentId))
    .innerJoin(schema.projects, eq(schema.projects.id, schema.alertEvents.projectId))
    .where(
      and(
        eq(schema.alertEvents.id, input.alertId),
        eq(schema.alertEvents.workspaceId, ctx.workspace.id),
      ),
    );

  if (!alert) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Alert not found" });
  }

  return alert;
});
