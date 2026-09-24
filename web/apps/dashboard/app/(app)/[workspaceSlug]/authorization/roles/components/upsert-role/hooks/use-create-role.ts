import { trpc } from "@/lib/trpc/client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import type { FormValues } from "../upsert-role.schema";

export const useCreateRole = (onSuccess: () => void) => {
  const trpcUtils = trpc.useUtils();
  return useMutation({
    mutationFn: async ({
      roleName,
      roleDescription,
      keyIds = [],
      permissionIds = [],
    }: FormValues) => {
      const unkey = getUnkeyClient();
      const permissions = await Promise.all(
        permissionIds.map(async (permission) => {
          const { data } = await unkey.permissions.getPermission({ permission });
          return data.slug;
        }),
      );
      await unkey.permissions.createRole({
        name: roleName,
        description: roleDescription,
        permissions,
      });
      const results = await Promise.allSettled(
        keyIds.map((keyId) => unkey.keys.addRoles({ keyId, roles: [roleName] })),
      );
      const failures = results.flatMap<unknown>((result) =>
        result.status === "rejected" ? [result.reason] : [],
      );
      return { keyCount: keyIds.length, failures };
    },
    onSuccess({ keyCount, failures }) {
      trpcUtils.authorization.invalidate();
      onSuccess();

      if (failures.length === 0) {
        toast.success("Role Created", { description: "Role created successfully" });
        return;
      }

      const { description } = getErrorToast(failures[0], "Failed to Assign Keys");
      toast.error(`Role Created, ${failures.length} of ${keyCount} Keys Not Assigned`, {
        description,
      });
    },
    onError(err) {
      const { message, description } = getErrorToast(err, "Failed to Save Role");
      toast.error(message, { description });
    },
  });
};
