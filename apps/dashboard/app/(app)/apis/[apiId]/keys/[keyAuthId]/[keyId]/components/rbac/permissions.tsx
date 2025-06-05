"use client";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { trpc } from "@/lib/trpc/client";
import { Checkbox } from "@unkey/ui";
import { toast } from "sonner";

export type Role = {
  id: string;
  name: string;
  isActive: boolean;
};

type PermissionTreeProps = {
  roles: Role[];
  keyId: string;
};

export function PermissionList({ roles, keyId }: PermissionTreeProps) {
  const trpcUtils = trpc.useUtils();

  const invalidatePermissions = () => {
    trpcUtils.key.fetchPermissions.invalidate();
  };

  const connectRole = trpc.rbac.connectRoleToKey.useMutation({
    onMutate: () => {
      toast.loading("Connecting role to key");
    },
    onSuccess: () => {
      toast.dismiss();
      toast.success("Role connected to key");

      invalidatePermissions();
    },
    onError: (error) => {
      toast.dismiss();
      toast.error(error.message);
    },
  });

  const disconnectRole = trpc.rbac.disconnectRoleFromKey.useMutation({
    onMutate: () => {
      toast.loading("Disconnecting role from key");
    },
    onSuccess: () => {
      toast.dismiss();
      toast.success("Role disconnected from key");

      invalidatePermissions();
    },
    onError: (error) => {
      toast.dismiss();
      toast.error(error.message);
    },
  });

  return (
    <Card className="flex flex-col flex-grow h-full min-h-[250px]">
      <CardHeader className="pb-0">
        <div className="mb-2">
          <CardTitle>Roles</CardTitle>
          <CardDescription>Manage roles for this key</CardDescription>
        </div>
      </CardHeader>
      <CardContent className="p-4">
        <div className="space-y-1">
          {roles.map((role) => (
            <div
              key={role.id}
              className="flex items-center space-x-2 p-2 rounded-lg hover:bg-gray-2"
            >
              <Checkbox
                checked={role.isActive}
                disabled={connectRole.isLoading || disconnectRole.isLoading}
                onCheckedChange={(checked) => {
                  if (checked) {
                    connectRole.mutate({ keyId: keyId, roleId: role.id });
                  } else {
                    disconnectRole.mutate({ keyId: keyId, roleId: role.id });
                  }
                }}
              />
              <div className="flex flex-col">
                <span className="text-sm font-medium">{role.name}</span>
              </div>
            </div>
          ))}

          {roles.length === 0 && (
            <div className="text-center py-4 text-sm text-accent-10">No roles available</div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
