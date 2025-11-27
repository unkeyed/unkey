"use client";;
import type { AuthenticatedUser, Membership, Organization } from "@/lib/auth/types";
import { useTRPC } from "@/lib/trpc/client";
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

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

type RoleSwitcherProps = {
  member: { id: string; role: string };
  organization: Organization;
  user: AuthenticatedUser;
  userMembership: Membership;
};

export const RoleSwitcher = memo<RoleSwitcherProps>(
  ({ member, organization, user, userMembership }) => {
    const trpc = useTRPC();
    const [role, setRole] = useState(member.role);
    const isAdmin = userMembership?.role === "admin";
    const queryClient = useQueryClient();

    const updateMember = useMutation(trpc.org.members.update.mutationOptions({
      onSuccess: () => {
        queryClient.invalidateQueries(trpc.org.members.list.pathFilter());
        toast.success("Role updated");
      },
      onError: (error) => {
        toast.error(error.message || "Failed to update role");
      },
    }));

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
        <div className="w-fit">
          <Select
            value={role}
            disabled={(Boolean(user) && member.id === user?.id) || updateMember.isPending}
            onValueChange={handleRoleUpdate}
          >
            <SelectTrigger className="w-[180px] max-sm:w-36">
              {updateMember.isPending ? <Loading /> : <SelectValue />}
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="basic_member">Member</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
      );
    }

    return <span className="text-content">{role === "admin" ? "Admin" : "Member"}</span>;
  },
);

RoleSwitcher.displayName = "RoleSwitcher";
