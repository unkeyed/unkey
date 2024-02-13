import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";
import { db, schema } from "@/lib/db";
import { auth } from "@clerk/nextjs";
import { newId } from "@unkey/id";
import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
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
  const { userId } = auth();

  if (props.searchParams.apiId) {
    const api = await db.query.apis.findFirst({
      where: (table, { eq }) => eq(table.id, props.searchParams.apiId!),
    });
    if (!api) {
      return notFound();
    }
    return (
      <div className="container m-16 mx-auto">
        <PageHeader
          title="Unkey"
          description="Create your first key"
          actions={[
            <Link
              key="skip"
              href="/app"
              className="flex items-center gap-1 text-sm duration-200 text-content-subtle hover:text-foreground"
            >
              Skip <ArrowRight className="w-4 h-4" />{" "}
            </Link>,
          ]}
        />

        <Separator className="my-6" />

        <Keys keyAuthId={api.keyAuthId!} apiId={api.id} />
      </div>
    );
  }
  if (props.searchParams.workspaceId) {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, props.searchParams.workspaceId!), isNull(table.deletedAt)),
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
              className="flex items-center gap-1 text-sm duration-200 text-content-subtle hover:text-foreground"
            >
              Skip <ArrowRight className="w-4 h-4" />{" "}
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
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, userId), isNull(table.deletedAt)),
    });

    // if no personal workspace exists, we create one
    if (!personalWorkspace) {
      const workspaceId = newId("workspace");
      await db.insert(schema.workspaces).values({
        id: workspaceId,
        tenantId: userId,
        name: "Personal",
        plan: "free",
        stripeCustomerId: null,
        stripeSubscriptionId: null,
        features: {},
        betaFeatures: {},
        subscriptions: null,
        createdAt: new Date(),
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
