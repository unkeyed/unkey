import { Metric } from "@/components/ui/metric";
import { clickhouse } from "@/lib/clickhouse";
import type { Key } from "@unkey/db";
import { Card, CardContent, CardHeader, CardTitle } from "@unkey/ui";
import { ArrowLeft } from "lucide-react";
import ms from "ms";
import Link from "next/link";
import { Suspense } from "react";

type Props = {
  params: {
    keyId: string;
    workspaceId: string;
  };
  rootKey: Key;
  children: React.ReactNode;
};

export function PageLayout({ children, rootKey: key, params: { keyId, workspaceId } }: Props) {
  return (
    <div className="flex flex-col gap-4">
      <Link
        href={`/${workspaceId}/settings/root-keys`}
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
          <Metric label="Created At" value={new Date(key.createdAtM).toDateString()} />
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

const LastUsed: React.FC<{
  workspaceId: string;
  keySpaceId: string;
  keyId: string;
}> = async ({ workspaceId, keySpaceId, keyId }) => {
  const lastUsed = await clickhouse.verifications
    .latest({ workspaceId, keySpaceId, keyId, limit: 1 })
    .then((res) => res.val?.at(0)?.time ?? 0);

  return (
    <Metric label="Last Used" value={lastUsed ? `${ms(Date.now() - lastUsed)} ago` : "Never"} />
  );
};
