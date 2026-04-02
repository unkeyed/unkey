"use server";
import { LoadingState } from "@/components/loading-state";
import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { ProjectsClient } from "./projects-client";

export default async function ProjectsPage() {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: { orgId, deletedAtM: { isNull: true } },
  });

  if (!workspace?.betaFeatures?.deployments) {
    // right now, we want to block all external access to deploy
    // to make it easier to opt-in for local development, comment out the redirect
    // and uncomment the <OptIn> component
    //return redirect("/apis");
    return notFound();
  }
  return (
    <Suspense fallback={<LoadingState message="Loading projects..." />}>
      <ProjectsClient />
    </Suspense>
  );
}
