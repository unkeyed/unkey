import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

import { BreadcrumbSkeleton } from "@/components/dashboard/breadcrumb-skeleton";
import { tags } from "@/lib/cache";
import { db } from "@/lib/db";
import { unstable_cache as cache } from "next/cache";
import { Suspense } from "react";
export const dynamic = "force-dynamic";
export const runtime = "edge";

type PageProps = {
  params: { permissionId: string };
};

async function AsyncPageBreadcrumb(props: PageProps) {
  const getPermissionById = cache(
    async (permissionId: string) =>
      db.query.permissions.findFirst({
        where: (table, { eq }) => eq(table.id, permissionId),
      }),
    ["permissionById"],
    { tags: [tags.permission(props.params.permissionId)] },
  );

  const permissions = await getPermissionById(props.params.permissionId);
  if (!permissions) {
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
          <BreadcrumbPage className="truncate w-96">{permissions.name}</BreadcrumbPage>
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
