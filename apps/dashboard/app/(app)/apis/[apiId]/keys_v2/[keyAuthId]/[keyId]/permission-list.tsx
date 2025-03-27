"use client";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { trpc } from "@/lib/trpc/client";
import { useRouter } from "next/navigation";
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
  const router = useRouter();
  const connectRole = trpc.rbac.connectRoleToKey.useMutation({
    onMutate: () => {
      toast.loading("Connecting role to key");
    },
    onSuccess: () => {
      toast.dismiss();
      toast.success("Role connected to key");
      router.refresh();
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
      router.refresh();
    },
    onError: (error) => {
      toast.dismiss();
      toast.error(error.message);
    },
  });

  return (
    <Card>
      <CardHeader className="pb-0">
        <div className="mb-2">
          <CardTitle>Roles</CardTitle>
          <CardDescription>Manage roles for this key</CardDescription>
        </div>
      </CardHeader>

      <CardContent className="pt-6">
        <div className="space-y-1">
          {roles.map((role) => (
            <div
              key={role.id}
              className="flex items-center space-x-2 p-2 rounded-lg hover:bg-gray-2"
            >
              <Checkbox
                checked={role.isActive}
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
        </div>
      </CardContent>
    </Card>
  );
}
