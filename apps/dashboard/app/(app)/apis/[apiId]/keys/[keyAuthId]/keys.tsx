import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import BackButton from "@/components/ui/back-button";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { db } from "@/lib/db";
import { ChevronRight, Scan, User, VenetianMask } from "lucide-react";
import Link from "next/link";

type Props = {
  keyAuthId: string;
  apiId: string;
};

export const dynamic = "force-dynamic";

export const Keys: React.FC<Props> = async ({ keyAuthId, apiId }) => {
  const keys = await db.query.keys.findMany({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.keyAuthId, keyAuthId), isNull(table.deletedAt)),
    limit: 100,
    with: {
      roles: true,
      permissions: true,
    },
  });

  const nullOwnerId = "UNKEY_NULL_OWNER_ID";
  type KeysByOwnerId = {
    [ownerId: string]: {
      id: string;
      keyAuthId: string;
      name: string | null;
      start: string | null;
      roles: number;
      permissions: number;
      environment: string | null;
    }[];
  };
  const keysByOwnerId = keys.reduce((acc, curr) => {
    const ownerId = curr.ownerId ?? nullOwnerId;
    if (!acc[ownerId]) {
      acc[ownerId] = [];
    }
    acc[ownerId].push({
      id: curr.id,
      keyAuthId: curr.keyAuthId,
      name: curr.name,
      start: curr.start,
      roles: curr.roles.length,
      permissions: curr.permissions.length,
      environment: curr.environment,
    });
    return acc;
  }, {} as KeysByOwnerId);

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      {keys.length === 0 ? (
        <EmptyPlaceholder>
          <EmptyPlaceholder.Icon>
            <Scan />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No keys found</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>Create your first key</EmptyPlaceholder.Description>
          <div className="flex flex-row">
            <CreateKeyButton apiId={apiId} keyAuthId={keyAuthId!} />
            <BackButton className="ml-4">Go Back</BackButton>
          </div>

          {/* <CreateNewRole trigger={<Button variant="primary">Create New Role</Button>} /> */}
        </EmptyPlaceholder>
      ) : (
        Object.entries(keysByOwnerId).map(([ownerId, ks]) => (
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-1">
              {ownerId === nullOwnerId ? (
                <div className="flex items-center justify-between gap-2 text-xs font-medium ph-no-capture">
                  <VenetianMask className="w-4 h-4 text-content" />
                  Without OwnerID
                  <span className="text-xs text-content-subtle">
                    You can associate keys with the a userId or other identifier from your own
                    system.
                  </span>
                </div>
              ) : (
                <div
                  key="apiId"
                  className="flex items-center justify-between gap-2 text-xs font-medium ph-no-capture"
                >
                  <User className="w-4 h-4 text-content" />
                  {ownerId}
                </div>
              )}
            </div>
            <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
              {ks.map((k) => (
                <Link
                  href={`/apis/${apiId}/keys/${k.keyAuthId}/${k.id}`}
                  key={k.id}
                  className="grid items-center sm:grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle gap-2"
                >
                  <div className="flex flex-col items-start col-span-6 ">
                    <span className="text-sm text-content">{k.name}</span>
                    <pre className="text-xs text-content-subtle">{k.id}</pre>
                  </div>

                  <div className="flex items-center col-span-2 gap-2">
                    {k.environment ? <Badge variant="secondary">env: {k.environment}</Badge> : null}
                  </div>

                  <div className="flex items-center col-span-3 gap-2">
                    <Badge variant="secondary">
                      {Intl.NumberFormat(undefined, { notation: "compact" }).format(k.permissions)}{" "}
                      Permission
                      {k.permissions !== 1 ? "s" : ""}
                    </Badge>

                    <Badge variant="secondary">
                      {Intl.NumberFormat(undefined, { notation: "compact" }).format(k.roles)} Role
                      {k.roles !== 1 ? "s" : ""}
                    </Badge>
                  </div>

                  <div className="flex items-center justify-end col-span-1">
                    <Button variant="ghost">
                      <ChevronRight className="w-4 h-4" />
                    </Button>
                  </div>
                </Link>
              ))}
            </ul>
          </div>
        ))
      )}
    </div>
  );
};
