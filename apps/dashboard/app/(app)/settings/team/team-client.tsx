"use client";
import Confirm from "@/components/dashboard/confirm";
import { Loading } from "@/components/dashboard/loading";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { PageContent } from "@/components/page-content";
import type { Workspace, Quotas } from "@unkey/db";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { toast } from "@/components/ui/toaster";
import { useOrganization, useUser } from "@/lib/auth/hooks";
import type {
  Invitation,
  InvitationListResponse,
  Membership,
  Organization,
  UpdateMembershipParams,
} from "@/lib/auth/types";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { useState, useCallback, memo, useMemo } from "react";
import { InviteButton } from "./invite";
import { User } from "@/lib/auth/types";
import Link from "next/link";
import { trpc } from "@/lib/trpc/client";

type MembersProps = {

  organization: Organization | null;
  user: User | null;
  userMembership: Membership | null;
};

type InvitationsProps = {
  user: User | null;
  organization: Organization | null;
};

type RoleSwitcherProps = {
  member: { id: string; role: string };
  organization: Organization | null;
  user: User | null;
  userMembership: Membership | null;
};

type StatusBadgeProps = {
  status: "pending" | "accepted" | "revoked" | "expired";
};

type Props = {
  workspace: Workspace & { quota: Quotas };
};

export const TeamPageClient: React.FC<Props> = ({ workspace }) => {
  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const { data: memberships, isLoading: isUserMembershipsLoading } = trpc.user.listMemberships.useQuery(user!.id, {
    enabled: !!user
  })
  const { data: organization, isLoading: isOrganizationLoading } = trpc.org.getOrg.useQuery(user!.orgId!, {
    enabled: !!user
  } );
  const userMemberships = memberships?.data;
  const currentOrgMembership = userMemberships?.find(
    (membership) => membership.organization.id === user?.orgId,
  );

  // Use useMemo for derived values
  const isAdmin = useMemo(() => {
    return currentOrgMembership?.role === "admin";
  }, [currentOrgMembership]);

  const isLoading = useMemo(() => {
    return isUserMembershipsLoading || isOrganizationLoading;
  }, [isUserMembershipsLoading, isOrganizationLoading]);

  type Tab = "members" | "invitations";
  const [tab, setTab] = useState<Tab>("members");


  const actions: React.ReactNode[] = [];

  if (isAdmin) {
    actions.push(
      <Select key="tab-select" value={tab} onValueChange={(value: Tab) => setTab(value)}>
        <SelectTrigger className="w-[180px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value="members">Members</SelectItem>
            <SelectItem value="invitations">Invitations</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>,
    );

    actions.push(
      <InviteButton
        key="invite-button"
        user={user!}
        organization={organization!}
      />,
    );
  }

  if (!workspace.quota.team) {
    return (
      <div className="w-full h-screen -mt-40 flex items-center justify-center">
        <Empty>
          <Empty.Title>Upgrade Your Plan to Add Team Members</Empty.Title>
          <Empty.Description>You can try it out for free for 14 days.</Empty.Description>
          <Empty.Actions>
            <Link href="/settings/billing">
              <Button>Upgrade</Button>
            </Link>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  return (
    <>
      <PageHeader title="Members" description="Manage your team members" actions={actions} />
      {isLoading ? (
        <Loading />
      ) : tab === "members" ? (
        <Members
          organization={organization!}
          user={user!}
          userMembership={currentOrgMembership!}
        />
      ) : (
        <Invitations
          organization={organization!}
          user={user!}
        />
      )}
    </>
  );
};

// Memoize components to prevent unnecessary re-renders
const Members = memo<MembersProps>(
  ({
    organization,
    user,
    userMembership,
  }) => {
    const { data: orgMemberships, isLoading } = trpc.org.members.list.useQuery(organization!.id);
    const memberships = orgMemberships!.data;
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
      }
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
          {isAdmin && (
            <InviteButton
              user={user}
              organization={organization}
            />
          )}
        </Empty>
      );
    }

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
                    <AvatarFallback>{member.fullName ?? member.email.slice(0, 1)}</AvatarFallback>
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
                    onConfirm={async () => {
                      try {
                        await removeMember.mutateAsync({ 
                          orgId: organization!.id, 
                          membershipId: id
                        });
                      } catch (error) {
                        console.error("Error removing member:", error);
                      }
                    }}
                    trigger={<Button>Remove</Button>}
                  />
                ) : null}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    );
  },
);

Members.displayName = "Members";

const Invitations = memo<InvitationsProps>(
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

const RoleSwitcher = memo<RoleSwitcherProps>(
  ({ member, organization, user, userMembership }) => {
    const [role, setRole] = useState(member.role);
    const isAdmin = userMembership?.role === "admin";
    const utils = trpc.useUtils();
    
    const updateMember = trpc.org.members.update.useMutation({
      onSuccess: () => {
        utils.org.members.list.invalidate();
        toast.success("Role updated");
      },
      onError: (error) => {
        toast.error(error.message || "Failed to update role");
      }
    });

    async function handleRoleUpdate(newRole: string) {
      if (!organization) {
        return;
      }
      
      try {
        await updateMember.mutateAsync({
          orgId: organization.id,
          membershipId: member.id,
          role: newRole,
        });

        setRole(newRole);
      } catch (err) {
        console.error(err);
      }
    }

    if (isAdmin) {
      return (
        <Select
          value={role}
          disabled={Boolean(user) && member.id === user?.id || updateMember.isLoading}
          onValueChange={handleRoleUpdate}
        >
          <SelectTrigger className="w-[180px] max-sm:w-36">
            {updateMember.isLoading ? <Loading /> : <SelectValue />}
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="admin">Admin</SelectItem>
              <SelectItem value="basic_member">Member</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      );
    }

    return <span className="text-content">{role === "admin" ? "Admin" : "Member"}</span>;
  }
);

RoleSwitcher.displayName = "RoleSwitcher";

RoleSwitcher.displayName = "RoleSwitcher";

const StatusBadge = memo<StatusBadgeProps>(({ status }) => {
  switch (status) {
    case "pending":
      return <Badge variant="primary">Pending</Badge>;
    case "accepted":
      return <Badge variant="secondary">Accepted</Badge>;
    case "revoked":
      return <Badge variant="secondary">Revoked</Badge>;
    case "expired":
      return <Badge variant="secondary">Expired</Badge>;
    default:
      return null;
  }
});

StatusBadge.displayName = "StatusBadge";
