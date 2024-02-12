import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { ChevronRight, Scan } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { CreateNewRole } from "./create-new-role";

export const revalidate = 0;

export default async function RolesPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      roles: {
        with: {
          keys: {
            columns: {
              keyId: true,
            },
          },
          permissions: {
            with: {
              permission: true,
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
    <div className="flex flex-col gap-8 mb-20 ">
      <div className="flex items-center justify-between flex-1 space-x-2">
        <h2 className="text-xl font-semibold text-content">Roles</h2>
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="h-8">
            {Intl.NumberFormat().format(workspace.roles.length)} /{" "}
            {Intl.NumberFormat().format(Number.POSITIVE_INFINITY)} used{" "}
          </Badge>
          <CreateNewRole trigger={<Button variant="primary">Create New Role</Button>} />
        </div>
      </div>

      {workspace.roles.length === 0 ? (
        <EmptyPlaceholder>
          <EmptyPlaceholder.Icon>
            <Scan />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No roles found</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>Create your first role</EmptyPlaceholder.Description>
          <CreateNewRole trigger={<Button variant="primary">Create New Role</Button>} />
        </EmptyPlaceholder>
      ) : (
        <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
          {workspace.roles.map((r) => (
            <Link
              href={`/app/authorization/roles/${r.id}`}
              key={r.id}
              className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle "
            >
              <div className="flex flex-col items-start col-span-6 ">
                <pre className="text-sm text-content">{r.name}</pre>
                <span className="text-xs text-content-subtle">{r.description}</span>
              </div>

              <div className="flex items-center col-span-3 gap-2">
                <Badge variant="secondary">
                  {Intl.NumberFormat(undefined, { notation: "compact" }).format(
                    r.permissions.length,
                  )}{" "}
                  Permission
                  {r.permissions.length !== 1 ? "s" : ""}
                </Badge>

                <Badge variant="secondary">
                  {Intl.NumberFormat(undefined, { notation: "compact" }).format(r.keys.length)} Key
                  {r.keys.length !== 1 ? "s" : ""}
                </Badge>
              </div>

              <div className="flex items-center justify-end col-span-3">
                <Button variant="ghost">
                  <ChevronRight className="w-4 h-4" />
                </Button>
              </div>
            </Link>
          ))}
        </ul>
      )}
    </div>
  );
}
