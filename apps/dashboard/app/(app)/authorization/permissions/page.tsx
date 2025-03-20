import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { asc, db } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { permissions } from "@unkey/db/src/schema";
import { Button } from "@unkey/ui";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { EmptyPermissions } from "./empty";
import { Navigation } from "./navigation";
export const revalidate = 0;
export default async function RolesPage() {
  const tenantId = getTenantId();

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

  const activeRoles = await db.query.roles.findMany({
    where: (table, { and, eq }) =>
      and(
        eq(table.workspaceId, workspace.id), // Use workspace ID from the fetched workspace
      ),
    columns: {
      id: true,
    },
  });

  const activeRoleIds = new Set(activeRoles.map((role) => role.id));

  /**
   * Filter out all the soft deleted keys and roles
   */
  workspace.permissions = workspace.permissions.map((permission) => {
    // Filter out deleted keys
    permission.keys = permission.keys.filter(({ key }) => key.deletedAtM === null);

    permission.roles = permission.roles.filter(
      ({ role }) => role?.id && activeRoleIds.has(role.id),
    );

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
              <EmptyPermissions />
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
                        {formatNumber(p.roles.length)} Role
                        {p.roles.length !== 1 ? "s" : ""}
                      </Badge>
                      <Badge variant="secondary">
                        {formatNumber(p.keys.length)} Key
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
