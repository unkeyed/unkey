"use client";

import { Confirm } from "@/components/dashboard/confirm";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { AuthenticatedUser, Membership, Organization } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import { Button, Empty, Loading, toast } from "@unkey/ui";
import { memo } from "react";
import { useTeamMembers } from "./hooks/use-team-members";
import { InviteButton } from "./invite";
import { RoleSwitcher } from "./role-switcher";

type MembersProps = {
  organization: Organization;
  user: AuthenticatedUser;
  userMembership: Membership;
};

export const MembersTanStack = memo<MembersProps>(({ organization, user, userMembership }) => {
  const {
    data: orgMemberships,
    isLoading,
    updateMemberRole,
    removeMember,
  } = useTeamMembers({
    orgId: organization?.id,
    enabled: !!organization?.id,
  });

  const memberships = orgMemberships?.data;
  const isAdmin = userMembership?.role === "admin";
  const utils = trpc.useUtils();

  const removeMemberMutation = trpc.org.members.remove.useMutation({
    onSuccess: () => {
      // Invalidate the member list query to trigger a refetch
      utils.org.members.list.invalidate();
      toast.success("Member removed successfully");
    },
    onError: (error) => {
      toast.error(error.message || "Failed to remove member");
    },
  });

  if (isLoading) {
    return (
      <div className="animate-in fade-in-50 relative flex min-h-[150px] flex-col items-center justify-center rounded-md border p-8 text-center">
        <Loading />
      </div>
    );
  }

  if (!memberships || memberships.length === 0) {
    return (
      <Empty>
        <Empty.Title>No team members</Empty.Title>
        <Empty.Description>Invite members to your team</Empty.Description>
        {isAdmin && <InviteButton user={user} organization={organization} />}
      </Empty>
    );
  }

  const handleRemoveMember = async (membershipId: string, orgId: string) => {
    try {
      // Use TanStack DB optimistic deletion
      removeMember(membershipId);

      // Also call the tRPC mutation for server synchronization
      await removeMemberMutation.mutateAsync({
        orgId,
        membershipId,
      });
    } catch (error) {
      console.error("Error removing member:", error);
      // TanStack DB will automatically rollback the optimistic deletion on error
    }
  };

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Member</TableHead>
          <TableHead>Role</TableHead>
          <TableHead>{/*/ empty */}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {memberships.map(({ id, role, user: member }) => (
          <TableRow key={id}>
            <TableCell>
              <div className="flex w-full items-center gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                <Avatar>
                  <AvatarImage src={member.avatarUrl ?? undefined} />
                  <AvatarFallback>
                    {member.fullName?.slice(0, 1) ?? member.email.slice(0, 1)}
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col items-start">
                  <span className="text-content font-medium">
                    {`${member.firstName ? member.firstName : member.email} ${
                      member.lastName ? member.lastName : ""
                    }`}
                  </span>
                  <span className="text-content-subtle text-xs">
                    {member.firstName ? member.email : ""}
                  </span>
                </div>
              </div>
            </TableCell>
            <TableCell>
              <RoleSwitcher
                member={{ id, role }}
                organization={organization}
                user={user}
                userMembership={userMembership}
              />
            </TableCell>
            <TableCell>
              {isAdmin && user && member.id !== user.id ? (
                <Confirm
                  variant="destructive"
                  title="Remove member"
                  description={`Are you sure you want to remove ${member.email}?`}
                  onConfirm={() => handleRemoveMember(id, organization.id)}
                  trigger={(onClick) => <Button onClick={onClick}>Remove</Button>}
                />
              ) : null}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
});

MembersTanStack.displayName = "MembersTanStack";
