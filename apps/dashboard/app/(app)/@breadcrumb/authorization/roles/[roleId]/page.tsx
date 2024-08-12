import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

import { BreadcrumbSkeleton } from "@/components/dashboard/breadcrumb-skeleton";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { unstable_cache as cache } from "next/cache";
import { Suspense } from "react";
export const dynamic = "force-dynamic";
export const runtime = "edge";

type PageProps = {
  params: { roleId: string };
};

async function AsyncPageBreadcrumb(props: PageProps) {
  const tenantId = getTenantId();
  const getWorkspaceByRoleId = cache(
    async (roleId: string) =>
      await db.query.workspaces.findFirst({
        where: (table, { eq }) => eq(table.tenantId, tenantId),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, roleId),
            with: {
              permissions: true,
            },
          },
          permissions: {
            with: {
              roles: true,
            },
            orderBy: (table, { asc }) => [asc(table.name)],
          },
        },
      }),
  );

  const workspace = await getWorkspaceByRoleId(props.params.roleId);
  if (!workspace) {
    return null;
  }

  const role = workspace.roles.at(0);
  if (!role) {
    return null;
  }
  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink href="/authorization">Authorization</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbLink href="/authorization/roles">Roles</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>{role.name}</BreadcrumbPage>
        </BreadcrumbItem>
      </BreadcrumbList>
    </Breadcrumb>
  );
}

export default function PageBreadcrumb(props: PageProps) {
  return (
    <Suspense fallback={<BreadcrumbSkeleton levels={2} />}>
      <AsyncPageBreadcrumb {...props} />
    </Suspense>
  );
}
