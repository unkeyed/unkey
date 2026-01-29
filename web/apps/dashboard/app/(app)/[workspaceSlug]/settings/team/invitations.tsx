"use client";

import type { Organization } from "@/lib/auth/types";
import { getGradientForUser } from "@/lib/avatar-gradient";
import { trpc } from "@/lib/trpc/client";
import { Card, CardContent } from "@unkey/ui";
import { Button, Empty, Loading, toast } from "@unkey/ui";
import { memo, useState } from "react";
import { StatusBadge } from "./status-badge";

type InvitationsProps = {
  organization: Organization;
  isAdmin: boolean;
};

export const Invitations = memo<InvitationsProps>(({ organization, isAdmin }) => {
  const { data: invitationsList, isLoading } = trpc.org.invitations.list.useQuery(organization.id);
  const invitations = invitationsList?.data;
  const utils = trpc.useUtils();
  const [revokingInvitationId, setRevokingInvitationId] = useState<string | null>(null);

  const revokeInvitation = trpc.org.invitations.remove.useMutation({
    onSuccess: () => {
      // Invalidate the invitation list query to trigger a refetch
      utils.org.invitations.list.invalidate();
      toast.success("Invitation revoked successfully");
    },
    onError: (error) => {
      toast.error(error.message || "Failed to revoke invitation");
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loading />
      </div>
    );
  }

  if (!invitations || invitations.length === 0) {
    return (
      <Empty>
        <Empty.Title>No pending invitations</Empty.Title>
        <Empty.Description>Invite members using the form above</Empty.Description>
      </Empty>
    );
  }

  return (
    <Card>
      <CardContent className="p-0">
        <div className="divide-y divide-border">
          {invitations.map((invitation) => (
            <div key={invitation.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <div
                  className="h-8 w-8 rounded-full flex-shrink-0"
                  style={{
                    background: `linear-gradient(to bottom right, ${getGradientForUser(invitation.email).from}, ${getGradientForUser(invitation.email).to})`,
                  }}
                />
                <div className="flex flex-col min-w-0 flex-1">
                  <span className="text-sm text-content truncate secret">{invitation.email}</span>
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
      </CardContent>
    </Card>
  );
});

Invitations.displayName = "Invitations";
