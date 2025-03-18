"use client";

import { revalidateTag } from "@/app/actions";
import { SettingCard } from "@/components/settings-card";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Clone } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
import { validation } from "@unkey/validation";
import { format } from "date-fns";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { DeleteRole } from "./delete-role";
import { Tree } from "./tree";

const formSchema = z.object({
  name: validation.name,
});

type FormValues = z.infer<typeof formSchema>;

export type NestedPermission = {
  id: string;
  checked: boolean;
  description: string | null;
  name: string;
  part: string;
  path: string;
  permissions: NestedPermissions;
};

export type NestedPermissions = Record<string, NestedPermission>;

type RoleClientProps = {
  role: {
    id: string;
    name: string;
    description?: string | null;
    createdAtM?: number;
    updatedAtM?: number | null;
    permissions: { permissionId: string }[];
  };
  activeKeys: { keyId: string }[];
  sortedNestedPermissions: NestedPermissions;
};

export const RoleClient = ({ role, activeKeys, sortedNestedPermissions }: RoleClientProps) => {
  const [isUpdating, setIsUpdating] = useState(false);
  const router = useRouter();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: role.name,
    },
  });

  const updateRoleMutation = trpc.rbac.updateRole.useMutation({
    onSuccess() {
      toast.success("Your role name has been updated!");
      revalidateTag(tags.role(role.id));
      router.refresh();
      setIsUpdating(false);
    },
    onError(err) {
      toast.error("Failed to update role name", {
        description: err.message,
      });
      setIsUpdating(false);
    },
  });

  const handleUpdateName = async (values: FormValues) => {
    const newName = values.name;

    if (newName === role.name) {
      return toast.error("Please provide a different name before saving.");
    }

    setIsUpdating(true);
    await updateRoleMutation.mutateAsync({
      description: role.description ?? "",
      id: role.id,
      name: newName,
    });
  };

  const watchedName = form.watch("name");

  const isNameChanged = watchedName !== role.name;
  const isNameValid = watchedName && watchedName.trim() !== "";

  // Get the count of active permissions for this role
  const activePermissionsCount = role.permissions.length;

  return (
    <div className="py-3 w-full flex items-center justify-center">
      <div className="w-[760px] flex flex-col justify-center items-center gap-5">
        <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          Role Settings
        </div>

        <div className="w-full">
          <form onSubmit={form.handleSubmit(handleUpdateName)}>
            <SettingCard
              title="Role name"
              description={
                <div>The name of this role used to identify it in API calls and the UI.</div>
              }
              border="top"
            >
              <div className="flex gap-2 items-center justify-center w-full">
                <Input placeholder="Role name" className="h-9" {...form.register("name")} />
                <Button
                  type="submit"
                  size="lg"
                  className="rounded-lg"
                  loading={isUpdating}
                  disabled={isUpdating || !isNameChanged || !isNameValid}
                >
                  Save
                </Button>
              </div>
            </SettingCard>
          </form>

          <SettingCard
            title="Role ID"
            description="An identifier for this role, used in API calls."
            border="bottom"
          >
            <Input
              readOnly
              disabled
              defaultValue={role.id}
              placeholder="Role ID"
              rightIcon={
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText(role.id);
                    toast.success("Copied to clipboard", {
                      description: role.id,
                    });
                  }}
                >
                  <Clone size="md-regular" />
                </button>
              }
            />
          </SettingCard>
        </div>

        <SettingCard
          title="Role Details"
          description="Information about this role"
          border="both"
          className="flex-col items-start"
          contentWidth="w-full"
        >
          <div className="grid grid-cols-2 gap-y-4 w-full">
            <div>
              <p className="text-sm text-accent-11">Created At</p>
              <p className="text-accent-12 font-medium text-sm">
                {role.createdAtM ? format(role.createdAtM, "PPPP") : "Unknown"}
              </p>
            </div>
            <div>
              <p className="text-sm text-accent-11">Updated At</p>
              <p className="text-accent-12 font-medium text-sm">
                {role.updatedAtM ? format(role.updatedAtM, "PPPP") : "Not updated yet"}
              </p>
            </div>
            <div>
              <p className="text-sm text-accent-11">Permissions</p>
              <p className="text-accent-12 font-medium text-sm">{activePermissionsCount}</p>
            </div>
            <div>
              <p className="text-sm text-accent-11">Connected Keys</p>
              <p className="text-accent-12 font-medium text-sm">{activeKeys.length}</p>
            </div>
          </div>
        </SettingCard>

        <SettingCard
          title="Role Permissions"
          description="Manage the permissions assigned to this role"
          border="both"
          className="flex-col items-start"
          contentWidth="w-full"
        >
          {Object.keys(sortedNestedPermissions).length > 0 ? (
            <Tree nestedPermissions={sortedNestedPermissions} role={{ id: role.id }} />
          ) : (
            <div className="w-full py-6 flex flex-col items-center justify-center text-center border border-dashed border-gray-4 rounded-lg bg-gray-1">
              <p className="text-sm text-accent-10 max-w-md">
                There are no permissions configured for this role. Permissions need to be created
                first before they can be assigned to roles.
              </p>
            </div>
          )}
        </SettingCard>

        <SettingCard
          title="Delete role"
          description={
            <>
              Deletes this role along with all its connections to permissions and keys. This action
              cannot be undone.
            </>
          }
          border="both"
        >
          <div className="w-full flex justify-end">
            <DeleteRole
              role={role}
              trigger={
                <Button className="w-fit rounded-lg" variant="outline" color="danger" size="lg">
                  Delete Role...
                </Button>
              }
            />
          </div>
        </SettingCard>
      </div>
    </div>
  );
};
