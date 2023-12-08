import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { QUOTA } from "@/lib/constants/quotas";
import { db, eq, schema } from "@/lib/db";
import { auth } from "@clerk/nextjs";
import { newId } from "@unkey/id";
import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { CreateApi } from "./create-api";
import { CreateWorkspace } from "./create-workspace";
import { Keys } from "./keys";

type Props = {
  searchParams: {
    workspaceId?: string;
    apiId?: string;
  };
};

export default async function (props: Props) {
  const _tenantId = getTenantId();
  const { userId } = auth();

  if (props.searchParams.apiId) {
    return (
      <div className="container m-16 mx-auto">
        <PageHeader
          title="Unkey"
          description="Create your first key"
          actions={[
            <Link
              key="skip"
              href="/app"
              className="text-content-subtle hover:text-foreground flex items-center gap-1 text-sm duration-200"
            >
              Skip <ArrowRight className="h-4 w-4" />{" "}
            </Link>,
          ]}
        />

        <Separator className="my-6" />

        <Keys apiId={props.searchParams.apiId} />
      </div>
    );
  }
  if (props.searchParams.workspaceId) {
    const workspace = await db.query.workspaces.findFirst({
      where: eq(schema.workspaces.id, props.searchParams.workspaceId),
    });
    if (!workspace) {
      return redirect("/new");
    }
    return (
      <div className="container m-16 mx-auto">
        <PageHeader
          title="Unkey"
          description="Create a new API"
          actions={[
            <Link
              key="skip"
              href="/app"
              className="text-content-subtle hover:text-foreground flex items-center gap-1 text-sm duration-200"
            >
              Skip <ArrowRight className="h-4 w-4" />{" "}
            </Link>,
          ]}
        />
        <Separator className="my-6" />
        <CreateApi workspace={workspace} />
      </div>
    );
  }

  if (userId) {
    const personalWorkspace = await db.query.workspaces.findFirst({
      where: eq(schema.workspaces.tenantId, userId),
    });

    // if no personal workspace exists, we create one
    if (!personalWorkspace) {
      const workspaceId = newId("workspace");
      await db.insert(schema.workspaces).values({
        id: workspaceId,
        slug: null,
        tenantId: userId,
        name: "Personal",
        plan: "free",
        stripeCustomerId: null,
        stripeSubscriptionId: null,
        maxActiveKeys: QUOTA.free.maxActiveKeys,
        maxVerifications: QUOTA.free.maxVerifications,
        usageActiveKeys: null,
        usageVerifications: null,
        lastUsageUpdate: null,
        billingPeriodStart: null,
        billingPeriodEnd: null,
        features: {},
        betaFeatures: {},
        subscriptions: null,
      });
      return redirect(`/new?workspaceId=${workspaceId}`);
    }
  }

  return (
    <div className="container m-16 mx-auto">
      <PageHeader title="Unkey" description="Create your workspace" />
      <Separator className="my-6" />
      <CreateWorkspace />
    </div>
  );
}
