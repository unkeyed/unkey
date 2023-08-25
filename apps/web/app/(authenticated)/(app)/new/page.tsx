import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { CreateWorkspace } from "./create-workspace";
import { CreateApi } from "./create-api";
import { Keys } from "./keys";
import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";
import Link from "next/link";
import { ArrowRight, ChevronRight } from "lucide-react";

type Props = {
  searchParams: {
    workspaceId?: string;
    apiId?: string;
  };
};

export default async function (props: Props) {
  const tenantId = getTenantId();

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
              className="flex items-center gap-1 text-sm duration-200 text-muted-foreground hover:text-foreground"
            >
              Skip <ArrowRight className="w-4 h-4" />{" "}
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
      return "Workspace not found";
    }
    return (
      <div className="container m-16 mx-auto">
        <PageHeader title="Unkey" description="Create a new API" />
        <Separator className="my-6" />
        <CreateApi workspace={workspace} />
      </div>
    );
  }

  const workspaces = await db.query.workspaces.findMany({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  return (
    <div className="container m-16 mx-auto">
      <PageHeader title="Unkey" description="Create your workspace" />
      <Separator className="my-6" />
      <CreateWorkspace workspaces={workspaces} />
    </div>
  );
}
