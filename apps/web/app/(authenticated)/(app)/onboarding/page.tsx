import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { redirect } from "next/navigation";
import { Onboarding } from "./client";
export default async function OnboardingPage() {
  const workspaceId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.id, workspaceId),
  });
  if (workspace) {
    redirect("/app");
  }

  return <Onboarding workspaceId={workspaceId} />;
}
