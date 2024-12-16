import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { ShieldKey } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { ChevronRight, Scan } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { CreateNewRole } from "./create-new-role";

export const revalidate = 0;

export default async function RolesPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      permissions: true,
      roles: {
        with: {
          keys: {
            with: {
              key: {
                columns: {
                  deletedAt: true,
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
    role.keys = role.keys.filter(({ key }) => key.deletedAt === null);
    return role;
  });

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<ShieldKey />}>
          <Navbar.Breadcrumbs.Link href="/authorization/roles">
            Authorization
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="/authorization/roles" active>
            Roles
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge variant="secondary" className="h-8">
            {Intl.NumberFormat().format(workspace.roles.length)} /{" "}
            {Intl.NumberFormat().format(Number.POSITIVE_INFINITY)} used{" "}
          </Badge>
          <CreateNewRole
            trigger={<Button variant="primary">Create New Role</Button>}
            permissions={workspace.permissions}
          />
        </Navbar.Actions>
      </Navbar>

      <PageContent>
        <SubMenu navigation={navigation} segment="roles" />
        <div className="mt-8 mb-20 overflow-x-auto">
          <div className="flex flex-col gap-8 mb-20 ">
            {workspace.roles.length === 0 ? (
              <EmptyPlaceholder>
                <EmptyPlaceholder.Icon>
                  <Scan />
                </EmptyPlaceholder.Icon>
                <EmptyPlaceholder.Title>No roles found</EmptyPlaceholder.Title>
                <EmptyPlaceholder.Description>Create your first role</EmptyPlaceholder.Description>
                <CreateNewRole
                  trigger={<Button variant="primary">Create New Role</Button>}
                  permissions={workspace.permissions}
                />
              </EmptyPlaceholder>
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
