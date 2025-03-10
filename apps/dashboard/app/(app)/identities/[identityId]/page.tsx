import { notFound } from "next/navigation";

import { CopyButton } from "@/components/dashboard/copy-button";
import { PageContent } from "@/components/page-content";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Code } from "@/components/ui/code";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { Button } from "@unkey/ui";
import { ChevronRight, Minus } from "lucide-react";
import ms from "ms";
import Link from "next/link";
import { Navigation } from "./navigation";
type Props = {
  params: {
    identityId: string;
  };
};

export default async function Page(props: Props) {
  const tenantId = await getTenantId();
  const identity = await db.query.identities.findFirst({
    where: (table, { eq }) => eq(table.id, props.params.identityId),
    with: {
      workspace: {
        columns: {
          tenantId: true,
        },
      },
      keys: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
        with: {
          keyAuth: {
            with: {
              api: true,
            },
          },
        },
      },
      ratelimits: true,
    },
  });

  if (!identity || identity.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div>
      <Navigation identityId={props.params.identityId} />
      <PageContent>
        <div className="flex flex-col gap-8">
          <div className="flex items-center justify-between gap-8">
            <div className="flex flex-col items-start gap-1 w-full">
              <span className="text-sm text-content-subtle whitespace-nowrap">Identity ID:</span>
              <Badge
                variant="secondary"
                className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
              >
                {identity.id}
                <CopyButton value={identity.id} />
              </Badge>
            </div>
            <div className="flex flex-col items-start gap-1 w-full">
              <span className="text-sm text-content-subtle whitespace-nowrap">External ID:</span>

              <Badge
                variant="secondary"
                className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
              >
                {identity.externalId}
                <CopyButton value={identity.externalId} />
              </Badge>
            </div>
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">Meta</h2>
          {identity.meta ? (
            <Code>{JSON.stringify(identity.meta, null, 2)}</Code>
          ) : (
            <Alert>
              <AlertTitle>No metadata</AlertTitle>
              <AlertDescription>This identity has no metadata.</AlertDescription>
            </Alert>
          )}

          <h2 className="text-2xl font-semibold tracking-tight">Ratelimits</h2>
          {identity.ratelimits.length === 0 ? (
            <Alert>
              <AlertTitle>No ratelimits</AlertTitle>
              <AlertDescription>This identity has no ratelimits attached.</AlertDescription>
            </Alert>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Limit</TableHead>
                  <TableHead>Duration</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {identity.ratelimits.map((ratelimit) => (
                  <TableRow>
                    <TableCell className="font-mono">{ratelimit.name}</TableCell>
                    <TableCell className="font-mono">
                      {Intl.NumberFormat(undefined, {
                        notation: "compact",
                      }).format(ratelimit.limit)}
                    </TableCell>
                    <TableCell className="font-mono">{ms(ratelimit.duration)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <h2 className="text-2xl font-semibold tracking-tight">Keys</h2>
          {identity.keys.length === 0 ? (
            <Alert>
              <AlertTitle>No keys</AlertTitle>
              <AlertDescription>This identity has no keys attached.</AlertDescription>
            </Alert>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Meta</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {identity.keys.map((key) => (
                  <TableRow>
                    <TableCell className="font-mono">{key.id}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {key.meta ? (
                        JSON.stringify(JSON.parse(key.meta), null, 2)
                      ) : (
                        <Minus className="text-content-subtle w-4 h-4" />
                      )}
                    </TableCell>
                    <LastUsed
                      workspaceId={key.workspaceId}
                      keySpaceId={key.keyAuthId}
                      keyId={key.id}
                    />
                    <TableCell className="flex justify-end">
                      <Link href={`/apis/${key.keyAuth.api.id}/keys/${key.keyAuth.id}/${key.id}`}>
                        <Button variant="ghost" shape="square">
                          <ChevronRight />
                        </Button>
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </PageContent>
    </div>
  );
}

const LastUsed: React.FC<{
  workspaceId: string;
  keySpaceId: string;
  keyId: string;
}> = async (props) => {
  const lastUsed = await clickhouse.verifications
    .latest({
      workspaceId: props.workspaceId,
      keySpaceId: props.keySpaceId,
      keyId: props.keyId,
    })
    .then((res) => res.val?.at(0)?.time ?? null);

  return (
    <TableCell>
      {lastUsed ? (
        <div className="flex items-center gap-4">
          <span className="text-content-subtle">{new Date(lastUsed).toUTCString()}</span>
          <span className="text-content">({ms(Date.now() - lastUsed)} ago)</span>
        </div>
      ) : (
        <Minus />
      )}
    </TableCell>
  );
};
