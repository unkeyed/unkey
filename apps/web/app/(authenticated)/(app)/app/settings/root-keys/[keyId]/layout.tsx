import { Card, CardContent } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { getLastUsed } from "@/lib/tinybird";
import ms from "ms";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { Selector } from "./selector";

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
      <Card>
        <CardContent className="grid grid-cols-5 divide-x">
          <Metric label="ID" value={<span className="font-mono">{key.id}</span>} />
          <Metric label="Name" value={key.name ?? "-"} />
          <Metric label="Created At" value={key.createdAt.toDateString()} />
          <Metric
            label={key.expires && key.expires.getTime() < Date.now() ? "Expired" : "Expires"}
            value={key.expires ? ms(key.expires.getTime() - Date.now()) : "-"}
          />

          <Suspense fallback={<div>x</div>}>
            <LastUsed keyId={keyId} />
          </Suspense>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Selector keyId={keyId} />
      </div>

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

const LastUsed: React.FC<{ keyId: string }> = async ({ keyId }) => {
  const lastUsed = await getLastUsed({ keyId }).then((res) => res.data.at(0)?.lastUsed ?? 0);

  return (
    <Metric label="Last Used" value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : "Never"} />
  );
};
