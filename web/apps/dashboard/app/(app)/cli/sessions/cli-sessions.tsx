"use client";

import type { User } from "@/lib/auth/types";
import { type CliSession, listCliSessions, revokeCliSession } from "@/lib/cli/device-api";
import { getErrorMessage } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, TimestampInfo, toast } from "@unkey/ui";

export type CliSessionOwner = Pick<User, "id" | "fullName" | "email">;

export function CliSessions({ owners }: { owners: CliSessionOwner[] }) {
  const { user } = useWorkspace();
  const ownersById = new Map(owners.map((owner) => [owner.id, owner]));
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["cli-sessions", user?.orgId],
    queryFn: listCliSessions,
  });
  const revoke = useMutation({
    mutationFn: revokeCliSession,
    async onSuccess() {
      toast.success("CLI session revoked.");
      await queryClient.invalidateQueries({ queryKey: ["cli-sessions"] });
    },
    onError(error) {
      toast.error(getErrorMessage(error, "Could not revoke this session."));
    },
  });

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 p-8">
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-lg text-gray-12">CLI sessions</h1>
        <p className="text-sm leading-5 text-gray-11">
          {user?.role === "admin"
            ? "You can see every CLI session in this workspace."
            : "You can see and revoke your own CLI sessions."}
        </p>
      </div>
      {query.isLoading ? <p className="text-sm text-gray-11">Loading sessions...</p> : null}
      {query.isError ? (
        <p className="text-sm text-gray-11">
          {getErrorMessage(query.error, "Could not load CLI sessions.")}
        </p>
      ) : null}
      {query.data && query.data.length === 0 ? (
        <p className="text-sm text-gray-11">No CLI sessions yet.</p>
      ) : null}
      <ul className="flex flex-col gap-3">
        {query.data?.map((session) => (
          <SessionRow
            key={session.id}
            session={session}
            showUser={user?.role === "admin"}
            owner={ownersById.get(session.userId)}
            revoking={revoke.isLoading && revoke.variables === session.id}
            onRevoke={() => revoke.mutate(session.id)}
          />
        ))}
      </ul>
    </div>
  );
}

function SessionRow({
  session,
  showUser,
  owner,
  revoking,
  onRevoke,
}: {
  session: CliSession;
  showUser: boolean;
  owner: CliSessionOwner | undefined;
  revoking: boolean;
  onRevoke: () => void;
}) {
  return (
    <li className="flex flex-col gap-2 rounded-md border p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 flex-col gap-1 text-sm">
          <span className="font-medium text-gray-12">{session.deviceName || "Unknown device"}</span>
          {showUser ? (
            <>
              <span className="break-words text-gray-11 secret">
                {owner?.fullName || owner?.email || "User unavailable"}
              </span>
              {owner?.fullName ? (
                <span className="break-all text-xs text-gray-11 secret">{owner.email}</span>
              ) : null}
            </>
          ) : null}
          <span className="text-gray-11">
            Started <TimestampInfo value={session.createdAt} displayType="relative" />
            {" · "}
            Last used <TimestampInfo value={session.lastUsedAt} displayType="relative" />
          </span>
          <span className="text-gray-11">
            {session.revokedAt ? (
              <>
                Revoked <TimestampInfo value={session.revokedAt} displayType="relative" />
              </>
            ) : session.expiresAt <= Date.now() ? (
              <>
                Expired <TimestampInfo value={session.expiresAt} displayType="relative" />
              </>
            ) : (
              <>
                Expires <TimestampInfo value={session.expiresAt} displayType="relative" />
              </>
            )}
          </span>
        </div>
        {session.revokedAt || session.expiresAt <= Date.now() ? null : (
          <Button type="button" variant="outline" loading={revoking} onClick={onRevoke}>
            Revoke
          </Button>
        )}
      </div>
      <p className="font-mono text-xs text-gray-11">
        {session.permissions.length} permission{session.permissions.length === 1 ? "" : "s"}
      </p>
    </li>
  );
}
