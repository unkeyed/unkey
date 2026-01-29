"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import type { AuthenticatedUser, Membership, Organization } from "@/lib/auth/types";
import { getGradientForUser } from "@/lib/avatar-gradient";
import { trpc } from "@/lib/trpc/client";
import { Card, CardContent } from "@unkey/ui";
import { Button, ConfirmPopover, Empty, Loading, toast } from "@unkey/ui";
import { memo, useMemo, useState } from "react";
import { RoleSwitcher } from "./role-switcher";

type MembersProps = {
  organization: Organization;
  user: AuthenticatedUser;
  userMembership: Membership;
};

export const Members = memo<MembersProps>(({ organization, user, userMembership }) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [currentMembership, setCurrentMembership] = useState<Membership | null>(null);
  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);
  const { data: orgMemberships, isLoading } = trpc.org.members.list.useQuery(organization?.id);
  const memberships = orgMemberships?.data;
  const isAdmin = userMembership?.role === "admin";
  const utils = trpc.useUtils();

  const anchorRef = useMemo(() => ({ current: anchorEl }), [anchorEl]);

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
      <div className="flex items-center justify-center py-12">
        <Loading />
      </div>
    );
  }

  const handleDeleteButtonClick = (
    membership: Membership,
    event: React.MouseEvent<HTMLButtonElement>,
  ) => {
    setAnchorEl(event.currentTarget);
    setCurrentMembership(membership);
    setIsConfirmPopoverOpen(true);
  };

  if (!memberships || memberships.length === 0) {
    return (
      <Empty>
        <Empty.Title>No team members</Empty.Title>
        <Empty.Description>Invite members using the form above</Empty.Description>
      </Empty>
    );
  }

  return (
    <>
      {currentMembership && anchorEl ? (
        <ConfirmPopover
          isOpen={isConfirmPopoverOpen}
          onOpenChange={setIsConfirmPopoverOpen}
          triggerRef={anchorRef}
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

      <Card>
        <CardContent className="p-0">
          <div className="divide-y divide-border">
            {memberships.map((membership) => {
              const { id, role, user: member } = membership;
              const isCurrentUser = member.id === user.id;

              return (
                <div key={id} className="flex items-center justify-between px-4 py-3">
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <Avatar className="h-8 w-8">
                      {member.avatarUrl && <AvatarImage src={member.avatarUrl} />}
                      <AvatarFallback
                        style={{
                          background: `linear-gradient(to bottom right, ${getGradientForUser(member.email).from}, ${getGradientForUser(member.email).to})`,
                        }}
                      />
                    </Avatar>
                    <div className="flex flex-col min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm text-content truncate secret">
                          {member.firstName
                            ? `${member.firstName}${member.lastName ? ` ${member.lastName}` : ""}`
                            : member.email}
                        </span>
                      </div>
                      {member.firstName && (
                        <span className="text-sm text-content-subtle truncate secret">{member.email}</span>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-3 ml-4">
                    <RoleSwitcher
                      member={{ id, role, userId: member.id }}
                      organization={organization}
                      user={user}
                      userMembership={userMembership}
                    />
                    {isAdmin && !isCurrentUser && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(event) => handleDeleteButtonClick(membership, event)}
                      >
                        Remove
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>
    </>
  );
});

Members.displayName = "Members";
