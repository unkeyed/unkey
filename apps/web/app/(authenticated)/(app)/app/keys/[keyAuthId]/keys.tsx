import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { db } from "@/lib/db";
import { DropdownMenu } from "@radix-ui/react-dropdown-menu";
import { TooltipContent, TooltipTrigger } from "@radix-ui/react-tooltip";
import { Key } from "@unkey/db";
import {
  Check,
  ChevronRight,
  FileClock,
  Minus,
  MoreHorizontal,
  MoreVertical,
  Scan,
  Trash,
  User,
  VenetianMask,
  X,
} from "lucide-react";
import ms from "ms";
import Link from "next/link";

type Props = {
  keyAuthId: string;
};

export const Keys: React.FC<Props> = async ({ keyAuthId }) => {
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
    });
    return acc;
  }, {} as KeysByOwnerId);

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <div className="flex items-center justify-between flex-1 space-x-2">
        <h2 className="text-xl font-semibold text-content">Keys</h2>
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="h-8">
            {Intl.NumberFormat().format(keys.length)} /{" "}
            {Intl.NumberFormat().format(Number.POSITIVE_INFINITY)} used{" "}
          </Badge>
          {/* <CreateNewRole trigger={<Button variant="primary">Create New Role</Button>} /> */}
        </div>
      </div>

      {keys.length === 0 ? (
        <EmptyPlaceholder>
          <EmptyPlaceholder.Icon>
            <Scan />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No keys found</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>Create your first key</EmptyPlaceholder.Description>
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
                  href={`/app/keys/${k.keyAuthId}/${k.id}`}
                  key={k.id}
                  className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle "
                >
                  <div className="flex flex-col items-start col-span-6 ">
                    <span className="text-sm text-content">{k.name}</span>
                    <pre className="text-xs text-content-subtle">{k.id}</pre>
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

                  <div className="flex items-center justify-end col-span-3">
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
