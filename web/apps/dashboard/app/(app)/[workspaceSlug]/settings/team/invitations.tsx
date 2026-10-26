"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import type { Organization } from "@/lib/auth/types";
import { listOf } from "@/lib/loadable/list";
import { fromQuery } from "@/lib/loadable/loadable";
import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import {
  Button,
  Card,
  CardContent,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  Loading,
  toast,
} from "@unkey/ui";
import { memo, useState } from "react";
import { StatusBadge } from "./status-badge";

type InvitationsProps = {
  organization: Organization;
  isAdmin: boolean;
};

export const Invitations = memo<InvitationsProps>(({ organization, isAdmin }) => {
  const view = listOf(
    fromQuery(trpc.org.invitations.list.useQuery(organization.id, { select: (res) => res.data })),
  );
  const utils = trpc.useUtils();
  const [revokingInvitationId, setRevokingInvitationId] = useState<string | null>(null);

  const revokeInvitation = trpc.org.invitations.remove.useMutation({
    onSuccess: () => {
      utils.org.invitations.list.invalidate();
      toast.success("Invitation revoked successfully");
    },
    onError: (error) => {
      toast.error(error.message || "Failed to revoke invitation");
    },
  });

  return match(view)
    .with({ status: "loading" }, () => (
      <div className="flex items-center justify-center py-12">
        <Loading />
      </div>
    ))
    .with({ status: "failed" }, ({ retry }) => (
      <Card>
        <CardContent className="p-0">
          <LoadError title="Could not load invitations" frame="none" onRetry={retry} />
        </CardContent>
      </Card>
    ))
    .with({ status: "empty" }, () => (
      <Card>
        <CardContent className="p-0">
          <EmptyState frame="none">
            <EmptyStateHeader>
              <EmptyStateTitle>No pending invitations</EmptyStateTitle>
              <EmptyStateDescription>Invite members using the form above</EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        </CardContent>
      </Card>
    ))
    .with({ status: "ready" }, ({ rows, footer }) => (
      <Card>
        <CardContent className="p-0">
          <div className="divide-y">
            {rows.map((invitation) => (
              <div key={invitation.id} className="flex items-center justify-between px-4 py-3">
                <div className="flex items-center gap-3 flex-1 min-w-0">
                  <Avatar className="w-8 h-8 rounded-full shrink-0">
                    <AvatarFallback name={invitation.email} />
                  </Avatar>
                  <div className="flex flex-col min-w-0 flex-1">
                    <span className="text-sm text-gray-12 truncate secret">{invitation.email}</span>
                    <div className="flex items-center">
                      <StatusBadge status={invitation.state} />
                    </div>
                  </div>
                </div>

                <div className="ml-4">
                  {invitation.state === "pending" && isAdmin && (
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={revokingInvitationId === invitation.id}
                      loading={revokingInvitationId === invitation.id}
                      onClick={async () => {
                        setRevokingInvitationId(invitation.id);
                        try {
                          await revokeInvitation.mutateAsync({
                            orgId: organization.id,
                            invitationId: invitation.id,
                          });
                        } catch (error) {
                          console.error("Error revoking invitation:", error);
                        } finally {
                          setRevokingInvitationId(null);
                        }
                      }}
                    >
                      Revoke
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
          <ListPager state={footer} noun="invitations" />
        </CardContent>
      </Card>
    ))
    .exhaustive();
});

Invitations.displayName = "Invitations";
