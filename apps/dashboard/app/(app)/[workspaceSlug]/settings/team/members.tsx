"use client";

import { ConfirmPopover } from "@/components/confirmation-popover";
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
import { memo, useRef, useState } from "react";
import { InviteButton } from "./invite";
import { RoleSwitcher } from "./role-switcher";

type MembersProps = {
  organization: Organization;
  user: AuthenticatedUser;
  userMembership: Membership;
};

export const Members = memo<MembersProps>(({ organization, user, userMembership }) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [currentMembership, setCurrentMembership] = useState<Membership | null>(null);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);
  const { data: orgMemberships, isLoading } = trpc.org.members.list.useQuery(organization?.id);
  const memberships = orgMemberships?.data;
  const isAdmin = userMembership?.role === "admin";
  const utils = trpc.useUtils();

  const removeMember = trpc.org.members.remove.useMutation({
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

  const handleDeleteButtonClick = (id: string) => {
    const foundMembership = memberships?.find((m) => m.id === id);
    setCurrentMembership(foundMembership ?? null);
    setIsConfirmPopoverOpen(true);
  };

  if (!memberships || memberships.length === 0) {
    return (
      <Empty>
        <Empty.Title>No team members</Empty.Title>
        <Empty.Description>Invite members to your team</Empty.Description>
        {isAdmin && <InviteButton user={user} organization={organization} />}
      </Empty>
    );
  }

  return (
    <>
      {currentMembership ? (
        <ConfirmPopover
          isOpen={isConfirmPopoverOpen}
          onOpenChange={setIsConfirmPopoverOpen}
          triggerRef={deleteButtonRef}
          description={`Are you sure you want to remove ${currentMembership.user.email}?`}
          confirmButtonText="Delete Member"
          cancelButtonText="Cancel"
          variant="danger"
          title="Remove member"
          onConfirm={async () => {
            try {
              await removeMember.mutateAsync({
                orgId: organization.id,
                membershipId: currentMembership.id,
              });
            } catch (error) {
              console.error("Error removing member:", error);
            }
          }}
        />
      ) : null}
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
                  <>
                    <Button
                      className="w-full"
                      type="button"
                      variant="default"
                      size="lg"
                      onClick={() => handleDeleteButtonClick(id)}
                      ref={deleteButtonRef}
                    >
                      Remove
                    </Button>
                  </>
                ) : null}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </>
  );
});

Members.displayName = "Members";
