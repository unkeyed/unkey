"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import type { AuthenticatedUser, Membership, Organization } from "@/lib/auth/types";
import { type ListFooterState, listOf } from "@/lib/loadable/list";
import { fromQuery } from "@/lib/loadable/loadable";
import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import {
  Button,
  Card,
  CardContent,
  ConfirmPopover,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  Loading,
  toast,
} from "@unkey/ui";
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
  const view = listOf(
    fromQuery(trpc.org.members.list.useQuery(organization?.id, { select: (res) => res.data })),
  );
  const utils = trpc.useUtils();

  const anchorRef = useMemo(() => ({ current: anchorEl }), [anchorEl]);

  const removeMember = trpc.org.members.remove.useMutation({
    onSuccess: () => {
      utils.org.members.list.invalidate();
      toast.success("Member removed successfully");
    },
    onError: (error) => {
      toast.error(error.message || "Failed to remove member");
    },
  });

  const handleDeleteButtonClick = (
    membership: Membership,
    event: React.MouseEvent<HTMLButtonElement>,
  ) => {
    setAnchorEl(event.currentTarget);
    setCurrentMembership(membership);
    setIsConfirmPopoverOpen(true);
  };

  return match(view)
    .with({ status: "loading" }, () => (
      <div className="flex items-center justify-center py-12">
        <Loading />
      </div>
    ))
    .with({ status: "failed" }, ({ retry }) => (
      <Card>
        <CardContent className="p-0">
          <LoadError title="Could not load team members" frame="none" onRetry={retry} />
        </CardContent>
      </Card>
    ))
    .with({ status: "empty" }, () => (
      <Card>
        <CardContent className="p-0">
          <EmptyState frame="none">
            <EmptyStateHeader>
              <EmptyStateTitle>No team members</EmptyStateTitle>
              <EmptyStateDescription>Invite members using the form above</EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        </CardContent>
      </Card>
    ))
    .with({ status: "ready" }, ({ rows, footer }) => (
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
        <MembersCard
          rows={rows}
          footer={footer}
          organization={organization}
          user={user}
          userMembership={userMembership}
          onRemove={handleDeleteButtonClick}
        />
      </>
    ))
    .exhaustive();
});

Members.displayName = "Members";

function MembersCard({
  rows,
  footer,
  organization,
  user,
  userMembership,
  onRemove,
}: MembersProps & {
  rows: readonly Membership[];
  footer: ListFooterState;
  onRemove: (membership: Membership, event: React.MouseEvent<HTMLButtonElement>) => void;
}) {
  const isAdmin = userMembership?.role === "admin";

  return (
    <Card>
      <CardContent className="p-0">
        <div className="divide-y">
          {rows.map((membership) => {
            const { id, role, user: member } = membership;
            const isCurrentUser = member.id === user.id;

            return (
              <div key={id} className="flex items-center justify-between px-4 py-3">
                <div className="flex items-center gap-3 flex-1 min-w-0">
                  <Avatar className="h-8 w-8">
                    {member.avatarUrl && <AvatarImage src={member.avatarUrl} />}
                    <AvatarFallback name={member.email} />
                  </Avatar>
                  <div className="flex flex-col min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-gray-12 truncate secret">
                        {member.firstName
                          ? `${member.firstName}${member.lastName ? ` ${member.lastName}` : ""}`
                          : member.email}
                      </span>
                    </div>
                    {member.firstName && (
                      <span className="text-sm text-gray-11 truncate secret">{member.email}</span>
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
                      onClick={(event) => onRemove(membership, event)}
                    >
                      Remove
                    </Button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
        <ListPager state={footer} noun="team members" />
      </CardContent>
    </Card>
  );
}
