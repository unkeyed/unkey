"use client";

import { revalidateTag } from "@/app/actions";
import { SettingCard } from "@/components/settings-card";
import { toast } from "@/components/ui/toaster";
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
import { z } from "zod";
import { DeletePermission } from "./delete-permission";

const formSchema = z.object({
  name: validation.name,
});

type FormValues = z.infer<typeof formSchema>;

type Props = {
  permission: {
    id: string;
    workspaceId: string;
    name: string;
    createdAtM: number;
    updatedAtM?: number | null;
    description?: string | null;
    keys: { keyId: string }[];
    roles?: {
      role?: {
        id?: string;
        deletedAt?: number | null;
        keys?: { keyId: string }[];
      };
    }[];
  };
};

export const PermissionClient = ({ permission }: Props) => {
  const [isUpdating, setIsUpdating] = useState(false);
  const router = useRouter();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: permission.name,
    },
  });

  // Filter out deleted or invalid roles
  const activeRoles = (permission?.roles ?? []).filter(
    (roleRelation) =>
      roleRelation?.role?.id &&
      (roleRelation.role.deletedAt === undefined || roleRelation.role.deletedAt === null),
  );

  // Count all unique connected keys
  const connectedKeys = new Set<string>();
  for (const key of permission.keys) {
    connectedKeys.add(key.keyId);
  }
  for (const roleRelation of activeRoles) {
    for (const key of roleRelation?.role?.keys ?? []) {
      connectedKeys.add(key.keyId);
    }
  }

  const updateNameMutation = trpc.rbac.updatePermission.useMutation({
    onSuccess() {
      toast.success("Your permission name has been updated!");
      revalidateTag(tags.permission(permission.id));
      router.refresh();
      setIsUpdating(false);
    },
    onError(err) {
      toast.error("Failed to update permission name", {
        description: err.message,
      });
      setIsUpdating(false);
    },
  });

  const handleUpdateName = async (values: FormValues) => {
    const newName = values.name;

    if (newName === permission.name) {
      return toast.error("Please provide a different name before saving.");
    }

    setIsUpdating(true);
    await updateNameMutation.mutateAsync({
      description: permission.description ?? "",
      id: permission.id,
      name: newName,
    });
  };

  const watchedName = form.watch("name");

  const isNameChanged = watchedName !== permission.name;
  const isNameValid = watchedName && watchedName.trim() !== "";

  return (
    <>
      <div className="py-3 w-full flex items-center justify-center ">
        <div className="w-[760px] flex flex-col justify-center items-center gap-5">
          <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
            Permission Settings
          </div>

          <div className="w-full">
            <form onSubmit={form.handleSubmit(handleUpdateName)}>
              <SettingCard
                title="Permission name"
                description={
                  <div>
                    Used in API calls. Changing this may affect your access control
                    <br /> requests.
                  </div>
                }
                border="top"
              >
                <div className="flex gap-2 items-center justify-center w-full">
                  <Input placeholder="Permission name" className="h-9" {...form.register("name")} />
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
              title="Permission ID"
              description="An identifier for the permission, used in API calls."
              border="bottom"
            >
              <Input
                readOnly
                disabled
                defaultValue={permission.id}
                placeholder="Permission ID"
                rightIcon={
                  <button
                    type="button"
                    onClick={() => {
                      navigator.clipboard.writeText(permission.id);
                      toast.success("Copied to clipboard", {
                        description: permission.id,
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
            title="Permission Details"
            description="Information about this permission"
            border="both"
            className="flex-col items-start"
            contentWidth="w-full"
          >
            <div className="grid grid-cols-2 gap-y-4 w-full">
              <div>
                <p className="text-sm text-accent-11">Created At</p>
                <p className="text-accent-12 font-medium text-sm">
                  {format(permission.createdAtM, "PPPP")}
                </p>
              </div>
              <div>
                <p className="text-sm text-accent-11">Updated At</p>
                <p className="text-accent-12 font-medium text-sm">
                  {permission.updatedAtM
                    ? format(permission.updatedAtM, "PPPP")
                    : "Not updated yet"}
                </p>
              </div>
              <div>
                <p className="text-sm text-accent-11">Connected Roles</p>
                <p className="text-accent-12 font-medium text-sm">{activeRoles.length}</p>
              </div>
              <div>
                <p className="text-sm text-accent-11">Connected Keys</p>
                <p className="text-accent-12 font-medium text-sm">{connectedKeys.size}</p>
              </div>
            </div>
          </SettingCard>

          <SettingCard
            title="Delete permission"
            description={
              <>
                Deletes this permission along with all its connections
                <br /> to roles and keys. This action cannot be undone.
              </>
            }
            border="both"
          >
            <div className="w-full flex justify-end">
              <DeletePermission
                permission={permission}
                trigger={
                  <Button className="w-fit rounded-lg" variant="outline" color="danger" size="lg">
                    Delete Permission...
                  </Button>
                }
              />
            </div>
          </SettingCard>
        </div>
      </div>
    </>
  );
};
