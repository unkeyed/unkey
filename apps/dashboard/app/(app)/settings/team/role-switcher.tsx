"use client";

import { Loading } from "@/components/dashboard/loading";
import { toast } from "@/components/ui/toaster";
import type { AuthenticatedUser, Membership, Organization } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { memo, useState } from "react";

type RoleSwitcherProps = {
  member: { id: string; role: string };
  organization: Organization;
  user: AuthenticatedUser;
  userMembership: Membership;
};

export const RoleSwitcher = memo<RoleSwitcherProps>(
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
      },
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
          disabled={(Boolean(user) && member.id === user?.id) || updateMember.isLoading}
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
  },
);

RoleSwitcher.displayName = "RoleSwitcher";
