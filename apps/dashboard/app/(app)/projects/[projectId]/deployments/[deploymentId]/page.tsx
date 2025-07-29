import { OptIn } from "@/components/opt-in";
import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { Suspense } from "react";

export default async function DeploymentsPage(): Promise<JSX.Element> {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  if (!workspace.betaFeatures.deployments) {
    // right now, we want to block all external access to deploy
    // to make it easier to opt-in for local development, comment out the redirect
    // and uncomment the <OptIn> component
    //return redirect("/apis");
    return <OptIn title="Projects" description="Projects are in beta" feature="deployments" />;
  }

  return (
    <Suspense fallback={<div>Loading...</div>}>
      <div>Deployment Details coming soon</div>
    </Suspense>
  );
}
