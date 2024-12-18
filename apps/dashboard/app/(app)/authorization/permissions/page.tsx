import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { asc, db } from "@/lib/db";
import { permissions } from "@unkey/db/src/schema";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { CreateNewPermission } from "./create-new-permission";
import { Navigation } from "./navigation";

export const revalidate = 0;

export default async function RolesPage() {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      permissions: {
        orderBy: [asc(permissions.name)],
        with: {
          keys: {
            with: {
              key: {
                columns: {
                  deletedAtM: true,
                },
              },
            },
          },
          roles: {
            with: {
              role: true,
            },
          },
        },
      },
    },
  });
  if (!workspace) {
    return redirect("/new");
  }

  /**
   * Filter out all the soft deleted keys cause I'm not smart enough to do it with drizzle
   */
  workspace.permissions = workspace.permissions.map((permission) => {
    permission.keys = permission.keys.filter(({ key }) => key.deletedAtM === null);
    return permission;
  });
  return (
    <div>
      <Navigation numberOfPermissions={workspace.permissions.length} />

      <PageContent>
        <SubMenu navigation={navigation} segment="permissions" />
        <div className="mt-8 mb-20 overflow-x-auto">
          <div className="flex items-center justify-between flex-1 space-x-2 w-full">
            {workspace.permissions.length === 0 ? (
              <Empty>
                <Empty.Icon />
                <Empty.Title>No permissions found</Empty.Title>
                <Empty.Description>Create your first permission</Empty.Description>
                <Empty.Actions>
                  <CreateNewPermission
                    trigger={<Button variant="primary">Create New Permission</Button>}
                  />
                </Empty.Actions>
              </Empty>
            ) : (
              <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border w-full">
                {workspace.permissions.map((p) => (
                  <Link
                    href={`/authorization/permissions/${p.id}`}
                    key={p.id}
                    className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle "
                  >
                    <div className="flex flex-col items-start col-span-6 ">
                      <pre className="text-sm text-content truncate w-full">{p.name}</pre>
                      <span className="text-xs text-content-subtle">{p.description}</span>
                    </div>

                    <div className="flex items-center col-span-3 gap-2">
                      <Badge variant="secondary">
                        {Intl.NumberFormat(undefined, {
                          notation: "compact",
                        }).format(p.roles.length)}{" "}
                        Role
                        {p.roles.length !== 1 ? "s" : ""}
                      </Badge>

                      <Badge variant="secondary">
                        {Intl.NumberFormat(undefined, {
                          notation: "compact",
                        }).format(p.keys.length)}{" "}
                        Key
                        {p.keys.length !== 1 ? "s" : ""}
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
        </div>
      </PageContent>
    </div>
  );
}
