"use server";
import { getAuthWithRedirect } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { ProjectsClient } from "./projects-client";

export default async function ProjectsPage() {
  const { orgId } = await getAuthWithRedirect();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace?.betaFeatures?.deployments) {
    // right now, we want to block all external access to deploy
    // to make it easier to opt-in for local development, comment out the redirect
    // and uncomment the <OptIn> component
    //return redirect("/apis");
    return notFound();
  }
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <ProjectsClient />
    </Suspense>
  );
}
