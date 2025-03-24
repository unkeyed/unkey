"use client";

import { memo } from "react";
import { trpc } from "@/lib/trpc/client";
import { Loading } from "@/components/dashboard/loading";
import { Empty } from "@unkey/ui";
import { toast } from "@/components/ui/toaster";
import { Button } from "@unkey/ui";
import { InviteButton } from "./invite";
import { StatusBadge } from "./status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Organization, User } from "@/lib/auth/types";

type InvitationsProps = {
  user: User | null;
  organization: Organization | null;
};

export const Invitations = memo<InvitationsProps>(
  ({ user, organization }) => {
    const { data: invitationsList, isLoading} = trpc.org.invitations.list.useQuery(organization!.id)
    const invitations = invitationsList!.data;
    const utils = trpc.useUtils();
    const revokeInvitation = trpc.org.invitations.remove.useMutation({
      onSuccess: () => {
      // Invalidate the invitation list query to trigger a refetch
      utils.org.invitations.list.invalidate();
      toast.success("Invitation revoked successfully");
    },
    onError: (error) => {
      toast.error(error.message || "Failed to revoke invitation");
    }})


    if (isLoading) {
      return (
        <div className="animate-in fade-in-50 relative flex min-h-[150px] flex-col items-center justify-center rounded-md border p-8 text-center">
          <Loading />
        </div>
      );
    }

    if (!invitations || invitations.length === 0) {
      return (
        <Empty>
          <Empty.Title>No pending invitations</Empty.Title>
          <Empty.Description>Invite members to your team</Empty.Description>
          <InviteButton
            user={user}
            organization={organization}
          />
        </Empty>
      );
    }

    return (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Email</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>{/*/ empty */}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {invitations.map((invitation) => (
            <TableRow key={invitation.id}>
              <TableCell>
                <span className="text-content font-medium">{invitation.email}</span>
              </TableCell>
              <TableCell>
                <StatusBadge status={invitation.state} />
              </TableCell>
              <TableCell>
                {invitation.state === "pending" && (
                  <Button
                    variant="destructive"
                    onClick={async () => {
                      try {
                        await revokeInvitation.mutateAsync({ 
                          orgId: organization!.id, 
                          invitationId: invitation.id
                        });
                      } catch (error) {
                        console.error("Error removing member:", error);
                      }
                    }}
                  >
                    Revoke
                  </Button>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    );
  },
);

Invitations.displayName = "Invitations";