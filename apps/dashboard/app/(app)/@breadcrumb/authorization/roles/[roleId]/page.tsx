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
  params: { roleId: string };
};

async function AsyncPageBreadcrumb(props: PageProps) {
  const getWorkspaceByRoleId = cache(
    async (roleId: string) =>
      await db.query.roles.findFirst({
        where: (table, { eq }) => eq(table.id, roleId),
      }),
    ["roleById"],
    { tags: [tags.role(props.params.roleId)] },
  );

  const role = await getWorkspaceByRoleId(props.params.roleId);
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
