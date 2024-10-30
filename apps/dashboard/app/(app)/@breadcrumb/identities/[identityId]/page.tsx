import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { unstable_cache as cache } from "next/cache";
import { Suspense } from "react";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type PageProps = {
  params: { identityId: string };
};

async function AsyncPageBreadcrumb(props: PageProps) {
  const tenantId = getTenantId();

  const getIdentityById = cache(
    async (identityId: string) =>
      db.query.identities.findFirst({
        where: (table, { eq }) => eq(table.id, identityId),

        with: {
          workspace: true,
        },
      }),
    ["identityById"],
  );

  const identity = await getIdentityById(props.params.identityId);
  if (!identity || identity.workspace.tenantId !== tenantId) {
    return null;
  }

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink href="/identities">Identities</BreadcrumbLink>
        </BreadcrumbItem>

        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>{identity.externalId}</BreadcrumbPage>
        </BreadcrumbItem>
      </BreadcrumbList>
    </Breadcrumb>
  );
}

export default function PageBreadcrumb(props: PageProps) {
  return (
    <Suspense fallback={null}>
      <AsyncPageBreadcrumb {...props} />
    </Suspense>
  );
}
