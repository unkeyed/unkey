"use client";
import Confirm from "@/components/dashboard/confirm";
import { Loading } from "@/components/dashboard/loading";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { PageContent } from "@/components/page-content";
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
import type { Invitation, Membership, Organization, UpdateMembershipParams } from "@/lib/auth/types";
import { Gear } from "@unkey/icons";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { useState, useCallback, memo } from "react";
import { navigation } from "../constants";
import { InviteButton } from "./invite";
import { User } from "@/lib/auth/types";
import { Navigation } from "@/components/navigation/navigation";

type MembersProps = {
  memberships: Membership[];
  loading: boolean;
  removeMember: (membershipId: string) => Promise<void>;
  updateMember: (params: UpdateMembershipParams) => Promise<void>;
  organization: Organization | null;
  user: User;
  userMembership: Membership
};

type InvitationsProps = {
  invitations: Invitation[];
  loading: boolean;
  revokeInvitation: (invitationId: string) => Promise<void>;
};

type RoleSwitcherProps = {
  member: { id: string; role: string };
  updateMember: (params: UpdateMembershipParams) => Promise<void>;
  organization: Organization | null;
  user: User;
  userMembership: Membership;
};

type StatusBadgeProps = {
  status: "pending" | "accepted" | "revoked" | "expired";
};

// Create a context to avoid prop drilling
export function TeamPageClient() {
  const userData = useUser();
  const orgData = useOrganization();

  const { membership } = userData;
  const isAdmin = membership?.role === "admin";
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

    actions.push(<InviteButton key="invite-button" />);
  }

  return (
    <div>
      <Navigation href="/settings/team" name="Settings" icon={<Gear />} />
      <PageContent>
        <SubMenu navigation={navigation} segment="team" />
        <div className="mb-20 flex flex-col gap-8 mt-8">
          <PageHeader title="Members" description="Manage your team members" actions={actions} />
          {tab === "members" ? (
            <Members 
              memberships={orgData.memberships} 
              loading={orgData.loading.memberships} 
              removeMember={orgData.removeMember}
              updateMember={orgData.updateMember}
              organization={orgData.organization}
              user={userData.user!}
              userMembership={membership!}
            />
          ) : (
            <Invitations 
              invitations={orgData.invitations} 
              loading={orgData.loading.invitations} 
              revokeInvitation={orgData.revokeInvitation}
            />
          )}
        </div>
      </PageContent>
    </div>
  );
}

// Memoize components to prevent unnecessary re-renders
const Members = memo<MembersProps>(({ memberships, loading, removeMember, updateMember, organization, user, userMembership }) => {
  const isAdmin = userMembership?.role === "admin";


  if (loading) {
    return (
      <div className="animate-in fade-in-50 relative flex min-h-[150px] flex-col items-center justify-center rounded-md border p-8 text-center">
        <Loading />
      </div>
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
        {memberships?.map(({ id, role, user: member }) => (
          <TableRow key={id}>
            <TableCell>
              <div className="flex w-full items-center gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                <Avatar>
                  <AvatarImage src={member.avatarUrl ?? undefined} />
                  <AvatarFallback>{member.fullName ?? member.email.slice(0, 2)}</AvatarFallback>
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
                updateMember={updateMember}
                organization={organization}
                user={user}
                userMembership={userMembership}
              />
            </TableCell>
            <TableCell>
              {isAdmin && member.id !== user?.id ? (
                <Confirm
                  variant="destructive"
                  title="Remove member"
                  description={`Are you sure you want to remove ${member.email}?`}
                  onConfirm={async () => {
                    if (member.id) {
                      await removeMember(id);
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
});

Members.displayName = 'Members';

const Invitations = memo<InvitationsProps>(({ invitations, loading, revokeInvitation }) => {
  if (loading) {
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
        <InviteButton />
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
        {invitations?.map((invitation) => (
          <TableRow key={invitation.id}>
            <TableCell>
              <span className="text-content font-medium">{invitation.email}</span>
            </TableCell>
            <TableCell>
              <StatusBadge status={invitation.state} />
            </TableCell>
            <TableCell>
              <Button
                variant="destructive"
                onClick={async () => {
                  await revokeInvitation(invitation.id);
                  toast.success("Invitation revoked");
                }}
              >
                Revoke
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
});

Invitations.displayName = 'Invitations';

const RoleSwitcher = memo<RoleSwitcherProps>(({ 
  member, 
  updateMember,
  organization,
  user,
  userMembership 
}) => {
  const [role, setRole] = useState(member.role);
  const [isLoading, setLoading] = useState(false);
  const isAdmin = userMembership?.role === "admin";

  const handleRoleUpdate = useCallback(async (newRole: string) => {
    try {
      setLoading(true);
      if (!organization) {
        return;
      }
      await updateMember({
        membershipId: member.id,
        role: newRole, // Fixed: was using member.role which doesn't change
      });

      setRole(newRole);
      toast.success("Role updated");
    } catch (err) {
      console.error(err);
      toast.error((err instanceof Error) ? err.message : "Failed to update role");
    } finally {
      setLoading(false);
    }
  }, [member.id, organization, updateMember]);

  if (isAdmin) {
    return (
      <Select
        value={role}
        disabled={member.id === user?.id}
        onValueChange={handleRoleUpdate}
      >
        <SelectTrigger className="w-[180px] max-sm:w-36">
          {isLoading ? <Loading /> : <SelectValue />}
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
});

RoleSwitcher.displayName = 'RoleSwitcher';

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

StatusBadge.displayName = 'StatusBadge';