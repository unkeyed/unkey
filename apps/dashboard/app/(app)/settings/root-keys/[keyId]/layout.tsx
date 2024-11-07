import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ArrowLeft } from "lucide-react";
import ms from "ms";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";

type Props = {
  params: {
    keyId: string;
  };
  children: React.ReactNode;
};

export default async function Layout({ children, params: { keyId } }: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return notFound();
  }

  const key = await db.query.keys.findFirst({
    where: (table, { eq, and }) => and(eq(table.forWorkspaceId, workspace.id), eq(table.id, keyId)),
  });
  if (!key) {
    return notFound();
  }

  return (
    <div className="flex flex-col gap-4">
      <Link
        href="/settings/root-keys"
        className="flex w-fit items-center gap-1 text-sm duration-200 text-content-subtle hover:text-secondary-foreground"
      >
        <ArrowLeft className="w-4 h-4" /> Back to Root Keys listing
      </Link>

      <Card>
        <CardHeader>
          <CardTitle>Root Key Information</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap justify-between divide-x [&>div:first-child]:pl-0">
          <Metric label="ID" value={<span className="font-mono">{key.id}</span>} />
          <Metric label="Created At" value={key.createdAt.toDateString()} />
          <Metric
            label={key.expires && key.expires.getTime() < Date.now() ? "Expired" : "Expires in"}
            value={key.expires ? ms(key.expires.getTime() - Date.now()) : "-"}
          />

          <Suspense fallback={<div>x</div>}>
            <LastUsed workspaceId={key.workspaceId} keySpaceId={key.keyAuthId} keyId={keyId} />
          </Suspense>
        </CardContent>
      </Card>

      {children}
    </div>
  );
}

const Metric: React.FC<{
  label: React.ReactNode;
  value: React.ReactNode;
}> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center px-4 py-2">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-sm leading-none tracking-tight ">{value}</div>
    </div>
  );
};

const LastUsed: React.FC<{ workspaceId: string; keySpaceId: string; keyId: string }> = async ({
  workspaceId,
  keySpaceId,
  keyId,
}) => {
  const lastUsed = await clickhouse.verifications
    .latest({ workspaceId, keySpaceId, keyId })
    .then((res) => res.at(0)?.time ?? 0);

  return (
    <Metric label="Last Used" value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : "Never"} />
  );
};
