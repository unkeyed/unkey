import { BackLink } from "@/components/back";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { UpdateCard } from "./settings";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  params: {
    namespaceId: string;
    overrideId: string;
  };
};

export default async function OverrideSettings(props: Props) {
  const tenantId = getTenantId();

  const override = await db.query.ratelimitOverrides.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(
        eq(table.namespaceId, props.params.namespaceId),
        eq(table.id, props.params.overrideId),
        isNull(schema.keys.deletedAt),
      ),
    with: {
      workspace: true,
    },
  });
  if (!override || override.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div className="flex flex-col gap-4">
      <BackLink href={`/app/ratelimits/${props.params.namespaceId}/overrides`} label="Back" />
      <Badge
        key="identifier"
        variant="secondary"
        className="flex justify-between gap-2 font-mono font-medium ph-no-capture"
      >
        {override.identifier}
        <CopyButton value={override.identifier} />
      </Badge>
      <UpdateCard
        overrideId={override.id}
        defaultValues={{
          limit: override.limit,
          duration: override.duration,
        }}
      />
    </div>
  );
}
