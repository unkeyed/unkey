"use client";

import { type OrganizationRole, isOrganizationRole, organizationRoleLabel } from "@/lib/auth/roles";
import type { AuthenticatedUser, Membership, Organization } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import {
  Loading,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { memo, useState } from "react";

type RoleSwitcherProps = {
  member: { id: string; role: string; userId: string };
  organization: Organization;
  user: AuthenticatedUser;
  userMembership: Membership;
};

export const RoleSwitcher = memo<RoleSwitcherProps>(
  ({ member, organization, user, userMembership }) => {
    const [role, setRole] = useState(member.role === "basic_member" ? "developer" : member.role);
    const isAdmin = userMembership?.role === "admin";
    const isCurrentUser = member.userId === user.id;
    const utils = trpc.useUtils();

    const updateMember = trpc.org.members.update.useMutation({
      onSuccess: () => {
        utils.org.members.list.invalidate();
        toast.success("Role updated");
      },
      onError: (error) => {
        toast.error(error.message || "Failed to update role");
      },
    });

    async function handleRoleUpdate(newRole: OrganizationRole) {
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
        <div className="w-fit">
          <Select
            value={role}
            items={[
              { value: "admin", label: "Admin" },
              { value: "developer", label: "Developer" },
              { value: "viewer", label: "Viewer" },
            ]}
            disabled={isCurrentUser || updateMember.isLoading}
            onValueChange={(newRole) => {
              if (isOrganizationRole(newRole)) {
                handleRoleUpdate(newRole);
              }
            }}
          >
            <SelectTrigger className="w-[180px] max-sm:w-36">
              {updateMember.isLoading ? <Loading /> : <SelectValue />}
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="developer">Developer</SelectItem>
                <SelectItem value="viewer">Viewer</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
      );
    }

    return <span className="text-content">{organizationRoleLabel(role)}</span>;
  },
);

RoleSwitcher.displayName = "RoleSwitcher";
