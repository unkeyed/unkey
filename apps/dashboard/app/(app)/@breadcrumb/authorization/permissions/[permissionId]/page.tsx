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
import { Suspense } from "react";
import { unstable_cache as cache } from "next/cache";
export const dynamic = "force-dynamic";
export const runtime = "edge";

type PageProps = {
  params: { permissionId: string };
};

async function AsyncPageBreadcrumb(props: PageProps) {
  const tenantId = getTenantId();
 const getWorkspaceByPermissionId = cache(
  async (permissionId : string) => db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      permissions: {
        where: (table, { eq }) => eq(table.id, permissionId),
        with: {
          keys: true,
          roles: {
            with: {
              role: {
                with: {
                  keys: {
                    columns: {
                      keyId: true,
                    },
                  },
                },
              },
            },
          },
        },
      },
    },
  }));

  const workspace = await getWorkspaceByPermissionId(props.params.permissionId);
  if (!workspace) {
    return null;
  }

  const permission = workspace.permissions.at(0);
  if (!permission) {
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
          <BreadcrumbLink href="/authorization/permissions">Permissions</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>{permission.name}</BreadcrumbPage>
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
