import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";
import { insertAuditLogs } from "@/lib/audit";
import { auth } from "@/lib/auth/server";
import { db, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { Button } from "@unkey/ui";
import { ArrowRight, GlobeLock, KeySquare } from "lucide-react";
import { headers } from "next/headers";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { CreateApi } from "./create-api";
import { CreateRatelimit } from "./create-ratelimit";
import { CreateWorkspace } from "./create-workspace";
import { RefreshHandler } from "./create-tenant/refresh-handler";
import { Keys } from "./keys";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: {
    workspaceId?: string;
    apiId?: string;
    ratelimitNamespaceId?: string;
    product?: "keys" | "ratelimit";
  };
};

export default async function (props: Props) {
  const user = await auth.getCurrentUser();
  // make typescript happy
  if (!user) {
    return redirect("/auth/sign-in");
  }

  const { id: userId, orgId } = user;

  // if they don't have an orgId, create one for them
  if (!orgId) {
    return redirect("/new/create-tenant");
  }

  if (props.searchParams.apiId) {
    const api = await db.query.apis.findFirst({
      where: (table, { eq }) => eq(table.id, props.searchParams.apiId!),
    });
    if (!api) {
      return notFound();
    }
    return (
      <div className="container m-16 mx-auto">
        <RefreshHandler />
        <PageHeader
          title="Unkey"
          description="Create your first key"
          actions={[
            <Link
              key="skip"
              href="/"
              className="flex items-center gap-1 text-sm duration-200 text-content-subtle hover:text-foreground"
            >
              Skip <ArrowRight className="w-4 h-4" />{" "}
            </Link>,
          ]}
        />

        <Separator className="my-8" />

        <Keys keyAuthId={api.keyAuthId!} apiId={api.id} />
      </div>
    );
  }
  if (props.searchParams.workspaceId && !props.searchParams.product) {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, props.searchParams.workspaceId!), isNull(table.deletedAtM)),
    });
    if (!workspace) {
      return redirect("/new");
    }
    return (
      <div className="container m-16 mx-auto">
        <RefreshHandler />
        <PageHeader
          title="Unkey"
          description="Choose your adventure"
          actions={[
            <Link
              key="skip"
              href="/"
              className="flex items-center gap-1 text-sm duration-200 text-content-subtle hover:text-foreground"
            >
              Skip <ArrowRight className="w-4 h-4" />{" "}
            </Link>,
          ]}
        />
        <Separator className="my-8" />
        <div className="grid grid-cols-1 gap-8 md::grid-cols-2">
          <div className="flex flex-col gap-4 lg:gap-10 p-8 duration-200 border rounded-lg border-border hover:border-primary justify-between">
            <div className="flex flex-col gap-4">
              <div className="flex items-center justify-center p-4 border rounded-lg bg-primary/5">
                <KeySquare className="w-6 h-6 text-primary" />
              </div>
              <h4 className="text-lg font-medium">I need API keys</h4>
              <p className="text-sm text-content-subtle">
                Create, verify, revoke keys for your public API.
              </p>
              <ol className="ml-2 space-y-1 text-sm list-disc list-outside text-content-subtle">
                <li>Globally distributed in 300+ locations</li>
                <li>Key and API analytics </li>
                <li>Scale to millions of requests</li>
              </ol>
            </div>

            <Link href={`/new?workspaceId=${workspace.id}&product=keys`}>
              <Button variant="primary" type="button" className="w-full">
                Create API
              </Button>
            </Link>
          </div>
          <div className="flex flex-col gap-4 lg:gap-10 p-8 duration-200 border rounded-lg border-border hover:border-primary justify-between">
            <div className="flex flex-col gap-4">
              <div className="flex items-center justify-center p-4 border rounded-lg bg-primary/5">
                <GlobeLock className="w-6 h-6 text-primary" />
              </div>
              <h4 className="text-lg font-medium">I want to ratelimit something</h4>
              <p className="text-sm text-content-subtle">
                Global low latency ratelimiting for your application.
              </p>
              <ol className="ml-2 space-y-1 text-sm list-disc list-outside text-content-subtle">
                <li>Low latency</li>
                <li>Globally consistent</li>
                <li>Powerful analytics</li>
              </ol>
            </div>
            <Link href={`/new?workspaceId=${workspace.id}&product=ratelimit`}>
              <Button variant="primary" type="button" className="w-full">
                Create Ratelimit
              </Button>
            </Link>
          </div>
        </div>
      </div>
    );
  }

  if (props.searchParams.product === "keys") {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, props.searchParams.workspaceId!), isNull(table.deletedAtM)),
    });
    if (!workspace) {
      return redirect("/new");
    }
    return (
      <div className="container m-16 mx-auto">
        <RefreshHandler />
        <PageHeader
          title="Unkey"
          description="Create your API"
          actions={[
            <Link
              key="skip"
              href="/"
              className="flex items-center gap-1 text-sm duration-200 text-content-subtle hover:text-foreground"
            >
              Skip <ArrowRight className="w-4 h-4" />{" "}
            </Link>,
          ]}
        />

        <Separator className="my-8" />

        <CreateApi workspace={workspace} />
      </div>
    );
  }
  if (props.searchParams.product === "ratelimit") {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, props.searchParams.workspaceId!), isNull(table.deletedAtM)),
      with: {
        auditLogBuckets: {
          where: (table, { eq }) => eq(table.name, "unkey_mutations"),
        },
      },
    });
    if (!workspace) {
      return redirect("/new");
    }
    return (
      <div className="container m-16 mx-auto">
        <RefreshHandler />
        <PageHeader
          title="Unkey"
          description="Create your ratelimit namespace"
          actions={[
            <Link
              key="skip"
              href="/"
              className="flex items-center gap-1 text-sm duration-200 text-content-subtle hover:text-foreground"
            >
              Skip <ArrowRight className="w-4 h-4" />{" "}
            </Link>,
          ]}
        />

        <Separator className="my-8" />

        <CreateRatelimit
          workspace={{ ...workspace, auditLogBucket: workspace.auditLogBuckets[0] }}
        />
      </div>
    );
  }
  if (orgId) {
    // do they already have a workspace?
    // they might if they have been invited to one
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, orgId), isNull(table.deletedAtM)),
    });

    // if no initial workspace exists, we create one
    if (!workspace) {
      const workspaceId = newId("workspace");
      await db.transaction(async (tx) => {
        await tx.insert(schema.workspaces).values({
          id: workspaceId,
          tenantId: orgId,
          name: "Personal",
          plan: "free",
          stripeCustomerId: null,
          stripeSubscriptionId: null,
          features: {},
          betaFeatures: {},
          subscriptions: null,
          createdAtM: Date.now(),
        });

          const bucketId = newId("auditLogBucket");
          await tx.insert(schema.auditLogBucket).values({
            id: bucketId,
            workspaceId,
            name: "unkey_mutations",
            retentionDays: 30,
            deleteProtection: true,
          });
  
          await insertAuditLogs(tx, bucketId, {
            workspaceId: workspaceId,
            event: "workspace.create",
            actor: {
              type: "user",
              id: userId,
            },
            description: `Created ${workspaceId}`,
            resources: [
              {
                type: "workspace",
                id: workspaceId,
              },
            ],

            context: {
              userAgent: headers().get("user-agent") ?? undefined,
              location: headers().get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
            },
          });
        });

      return redirect(`/new?workspaceId=${workspaceId}`);
    }
  }

  return (
    <div className="container m-16 mx-auto">
      <RefreshHandler />
      <PageHeader title="Unkey" description="Create your workspace" />
      <Separator className="my-8" />
      <CreateWorkspace />
    </div>
  );
}
