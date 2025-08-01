import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound, redirect } from "next/navigation";
import { Suspense } from "react";
import { ProjectsClient } from "./projects-client";

export default async function ProjectsPage(): Promise<JSX.Element> {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  if (!workspace.betaFeatures.deployments) {
    return notFound();
  }

  return (
    <Suspense fallback={<div>Loading...</div>}>
      <ProjectsClient />
    </Suspense>
  );
}
