"use client";

import { Label } from "@/components/ui/label";
import { trpc } from "@/lib/trpc/client";
import { Checkbox, toast } from "@unkey/ui";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
type Props = {
  rootKeyId: string;
  permissionName: string;
  label: string;
  description: string;
  checked: boolean;
  preventEnabling?: boolean;
  preventDisabling?: boolean;
};

export const PermissionToggle: React.FC<Props> = ({
  rootKeyId,
  permissionName,
  label,
  checked,
  description,
  preventEnabling,
  preventDisabling,
}) => {
  const router = useRouter();

  const [optimisticChecked, setOptimisticChecked] = useState(checked);
  const addPermission = trpc.rbac.addPermissionToRootKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(true);
    },
    onSuccess: () => {
      toast.success("Permission added", {
        description: "Changes may take up to 60 seconds to take effect.",
      });
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
    onSettled: () => {
      router.refresh();
    },
  });
  const removeRole = trpc.rbac.removePermissionFromRootKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(false);
    },
    onSuccess: () => {
      toast.success("Permission removed", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            addPermission.mutate({ rootKeyId, permission: permissionName });
          },
        },
      });
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
    onSettled: () => {
      router.refresh();
    },
  });

  return (
    <div className="@container flex items-center text-start">
      <div className="flex flex-col items-center @xl:flex-row w-full">
        <div className="w-full @xl:w-1/3">
          <div className="flex items-center gap-2">
            {addPermission.isLoading || removeRole.isLoading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Checkbox
                disabled={
                  addPermission.isLoading ||
                  removeRole.isLoading ||
                  (preventEnabling && !checked) ||
                  (preventDisabling && checked)
                }
                checked={optimisticChecked}
                onClick={() => {
                  if (checked) {
                    if (!preventDisabling) {
                      removeRole.mutate({ rootKeyId, permissionName });
                    }
                  } else {
                    if (!preventEnabling) {
                      addPermission.mutate({
                        rootKeyId,
                        permission: permissionName,
                      });
                    }
                  }
                }}
              />
            )}
            <Label className="text-xs text-content">{label}</Label>
          </div>
        </div>

        <p className="w-full text-xs text-content-subtle @xl:w-2/3 pl-8 pb-2 @xl:pb-0 @xl:mt-0">
          {description}
        </p>
      </div>
    </div>
  );
};
