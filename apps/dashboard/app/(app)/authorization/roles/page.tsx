import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { CreateNewRole } from "./create-new-role";
import { Navigation } from "./navigation";

export const revalidate = 0;

export default async function RolesPage() {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      permissions: true,
      roles: {
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

  /**
   * Filter out all the soft deleted keys cause I'm not smart enough to do it with drizzle
   */
  workspace.roles = workspace.roles.map((role) => {
    role.keys = role.keys.filter(({ key }) => key.deletedAtM === null);
    return role;
  });

  return (
    <div>
      <Navigation workspace={workspace} />

      <PageContent>
        <SubMenu navigation={navigation} segment="roles" />
        <div className="mt-8 mb-20 overflow-x-auto">
          <div className="flex flex-col gap-8 mb-20 ">
            {workspace.roles.length === 0 ? (
              <Empty>
                <Empty.Icon />
                <Empty.Title>No roles found</Empty.Title>
                <Empty.Description>Create your first role</Empty.Description>
                <Empty.Actions>
                  <CreateNewRole
                    trigger={<Button variant="primary">Create New Role</Button>}
                    permissions={workspace.permissions}
                  />
                </Empty.Actions>
              </Empty>
            ) : (
              <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
                {workspace.roles.map((r) => (
                  <Link
                    href={`/authorization/roles/${r.id}`}
                    key={r.id}
                    className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle "
                  >
                    <div className="flex flex-col items-start col-span-6 ">
                      <pre className="text-sm text-content truncate w-full pr-1">{r.name}</pre>
                      <span className="text-xs text-content-subtle truncate w-full">
                        {r.description}
                      </span>
                    </div>

                    <div className="flex items-center col-span-3 gap-2">
                      <Badge variant="secondary">
                        {Intl.NumberFormat(undefined, {
                          notation: "compact",
                        }).format(r.permissions.length)}{" "}
                        Permission
                        {r.permissions.length !== 1 ? "s" : ""}
                      </Badge>

                      <Badge variant="secondary">
                        {Intl.NumberFormat(undefined, {
                          notation: "compact",
                        }).format(r.keys.length)}{" "}
                        Key
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
        </div>
      </PageContent>
    </div>
  );
}
