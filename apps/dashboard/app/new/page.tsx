import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { insertAuditLogs } from "@/lib/audit";
import { serverAuth } from "@/lib/auth/server";
import { db, schema } from "@/lib/db";
import { ingestAuditLogsTinybird } from "@/lib/tinybird";
import { auth } from "@clerk/nextjs";
import { newId } from "@unkey/id";
import { ArrowRight, DatabaseZap, GlobeLock, KeySquare } from "lucide-react";
import { headers } from "next/headers";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { CreateApi } from "./create-api";
import { CreateRatelimit } from "./create-ratelimit";
import { CreateSemanticCacheButton } from "./create-semantic-cache";
import { CreateWorkspace } from "./create-workspace";
import { Keys } from "./keys";

type Props = {
  searchParams: {
    workspaceId?: string;
    apiId?: string;
    ratelimitNamespaceId?: string;
    product?: "keys" | "ratelimit";
  };
};

export default async function (props: Props) {
  const user = await serverAuth.getUser();

  if (!user) {
    return notFound();
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
        and(eq(table.id, props.searchParams.workspaceId!), isNull(table.deletedAt)),
    });
    if (!workspace) {
      return redirect("/new");
    }
    return (
      <div className="container m-16 mx-auto">
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
        <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
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
          <div className="flex flex-col gap-4 lg:gap-10 p-8 duration-200 border rounded-lg border-border hover:border-primary justify-between">
            <div className="flex flex-col gap-4">
              <div className="flex items-center justify-center p-4 border rounded-lg bg-primary/5">
                <DatabaseZap className="w-6 h-6 text-primary" />
              </div>
              <h4 className="text-lg font-medium">I want to cache an LLM</h4>
              <p className="text-sm text-content-subtle">
                Faster, cheaper LLM API calls through re-using semantically similar previous
                responses.
              </p>
              <ol className="ml-2 space-y-1 text-sm list-decimal list-outside text-content-subtle">
                <li>You switch out the baseUrl in your requests to OpenAI with your gateway URL</li>
                <li>Unkey will automatically start caching your responses</li>
                <li>Monitor and track your cache usage here</li>
              </ol>
            </div>
            <CreateSemanticCacheButton />
          </div>
        </div>
      </div>
    );
  }

  if (props.searchParams.product === "keys") {
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
        and(eq(table.id, props.searchParams.workspaceId!), isNull(table.deletedAt)),
    });
    if (!workspace) {
      return redirect("/new");
    }
    return (
      <div className="container m-16 mx-auto">
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

        <CreateRatelimit />
      </div>
    );
  }
  if (user.id) {
    const personalWorkspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, user.id), isNull(table.deletedAt)),
    });

    // if no personal workspace exists, we create one
    if (!personalWorkspace) {
      const workspaceId = newId("workspace");
      await db.transaction(async (tx) => {
        await tx
          .insert(schema.workspaces)
          .values({
            id: workspaceId,
            tenantId: user.id,
            name: "Personal",
            plan: "free",
            stripeCustomerId: null,
            stripeSubscriptionId: null,
            features: {},
            betaFeatures: {},
            subscriptions: null,
            createdAt: new Date(),
          })
          .onDuplicateKeyUpdate({ set: { updatedAt: new Date() } });
        await insertAuditLogs(tx, {
          workspaceId: workspaceId,
          event: "workspace.create",
          actor: {
            type: "user",
            id: user.id,
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
      await ingestAuditLogsTinybird({
        workspaceId: workspaceId,
        event: "workspace.create",
        actor: {
          type: "user",
          id: user.id,
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

      return redirect(`/new?workspaceId=${workspaceId}`);
    }
  }

  return (
    <div className="container m-16 mx-auto">
      <PageHeader title="Unkey" description="Create your workspace" />
      <Separator className="my-8" />
      <CreateWorkspace />
    </div>
  );
}
