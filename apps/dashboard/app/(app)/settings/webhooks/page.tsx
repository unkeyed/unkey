import { PageHeader } from "@/components/dashboard/page-header";

import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { BookOpen, ChevronRight, Scan, Webhook } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { CreateWebhookButton } from "./create-webhook-button";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RatelimitOverviewPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      webhooks: {
        with: {
          events: {
            where: (table, { gt }) => gt(table.time, Date.now() - 7 * 24 * 60 * 60 * 1000),
            columns: {},
            with: {
              deliveryAttempts: {
                columns: { success: true },
              },
            },
          },
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div>
      {workspace.webhooks.length > 0 ? (
        <div className="flex flex-col gap-8 mb-20 ">
          <PageHeader title="Webhooks" actions={[<CreateWebhookButton key="create-webhook" />]} />
          {workspace.webhooks.length === 0 ? (
            <EmptyPlaceholder>
              <EmptyPlaceholder.Icon>
                <Scan />
              </EmptyPlaceholder.Icon>
              <EmptyPlaceholder.Title>No webhooks found</EmptyPlaceholder.Title>
              <EmptyPlaceholder.Description>Create your first webhook</EmptyPlaceholder.Description>
              {/* <CreateNewRole trigger={<Button variant="primary">Create New Role</Button>} /> */}
            </EmptyPlaceholder>
          ) : (
            <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
              {workspace.webhooks.map((wh) => {
                const errors = wh.events
                  .flatMap((e) => e.deliveryAttempts)
                  .reduce((acc, attempt) => {
                    if (!attempt.success) {
                      return acc + 1;
                    }
                    return acc;
                  }, 0);

                return (
                  <Link
                    href={`/settings/webhooks/${wh.id}`}
                    key={wh.id}
                    className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle "
                  >
                    <div className="flex flex-col items-start col-span-7 ">
                      <span className="text-sm text-content">{wh.destination}</span>
                    </div>

                    <div className="flex items-center col-span-2 gap-2">
                      <Badge variant={errors > 0 ? "alert" : "secondary"}>{errors} Errors</Badge>
                    </div>
                    <div className="flex items-center col-span-2 gap-2">
                      <Badge variant={wh.enabled ? "primary" : "secondary"}>
                        {wh.enabled ? "Enabled" : "Disabled"}
                      </Badge>
                    </div>

                    <div className="flex items-center justify-end col-span-1">
                      <Button variant="ghost">
                        <ChevronRight className="w-4 h-4" />
                      </Button>
                    </div>
                  </Link>
                );
              })}
            </ul>
          )}
        </div>
      ) : (
        <EmptyPlaceholder className="my-4 ">
          <EmptyPlaceholder.Icon>
            <Webhook />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No Webhooks found</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>
            You haven&apos;t created any webhooks yet.
          </EmptyPlaceholder.Description>

          <div className="flex flex-col items-center justify-center gap-2 md:flex-row">
            <Link href="/docs" target="_blank">
              <Button variant="secondary" className="items-center w-full gap-2 ">
                <BookOpen className="w-4 h-4 " />
                Read the docs
              </Button>
            </Link>
          </div>
        </EmptyPlaceholder>
      )}
    </div>
  );
}
