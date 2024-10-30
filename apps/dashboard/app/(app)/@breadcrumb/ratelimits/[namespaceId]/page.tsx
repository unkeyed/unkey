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
  params: { namespaceId: string };
};

async function AsyncPageBreadcrumb(props: PageProps) {
  const getNamespaceById = cache(
    async (namespaceId: string) =>
      db.query.ratelimitNamespaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, namespaceId), isNull(table.deletedAt)),
      }),

    ["namespaceById"],
    { tags: [tags.namespace(props.params.namespaceId)] },
  );

  const namespace = await getNamespaceById(props.params.namespaceId);
  if (!namespace) {
    return null;
  }

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink href="/ratelimits">Ratelimits</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>{namespace.name}</BreadcrumbPage>
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
