import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { keys } from "@unkey/db/src/schema";
import { Button } from "@unkey/ui";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { EmptyRoles } from "./empty";
import { Navigation } from "./navigation";

export const revalidate = 0;

export default async function RolesPage() {
  const tenantId = getTenantId();

  // Get workspace with all permissions and roles with their permissions
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      permissions: true,
      roles: {
        with: {
          // Include all permissions for each role
          permissions: {
            with: {
              permission: true,
            },
          },
          // Only include non-deleted keys
          keys: {
            where: (keysRolesTable, { exists }) =>
              exists(
                db
                  .select()
                  .from(keys)
                  .where(and(eq(keys.id, keysRolesTable.keyId), isNull(keys.deletedAtM))),
              ),
            with: {
              key: true,
            },
          },
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const activePermissionIds = new Set(workspace.permissions.map((p) => p.id));

  const enhancedRoles = workspace.roles.map((role) => {
    const filteredPermissions = role.permissions.filter((rp) =>
      activePermissionIds.has(rp.permissionId),
    );

    return {
      ...role,
      // Replace the permissions array with filtered one
      permissions: filteredPermissions,
      // Add permission count for display
      permissionCount: filteredPermissions.length,
    };
  });

  // Create the final workspace object with enhanced roles
  const workspaceWithRoles = {
    ...workspace,
    roles: enhancedRoles,
  };

  return (
    <div>
      <Navigation workspace={workspaceWithRoles} />
      <PageContent>
        <SubMenu navigation={navigation} segment="roles" />
        <div className="mt-8 mb-20 overflow-x-auto">
          <div className="flex flex-col gap-8 mb-20 ">
            {workspaceWithRoles.roles.length === 0 ? (
              <EmptyRoles />
            ) : (
              <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
                {workspaceWithRoles.roles.map((r) => (
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
                        {formatNumber(r.permissionCount)} Permissions
                      </Badge>
                      <Badge variant="secondary">
                        {formatNumber(r.keys.length)} Key
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
