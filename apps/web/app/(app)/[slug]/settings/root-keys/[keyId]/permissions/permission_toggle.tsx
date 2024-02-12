"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { trpc } from "@/lib/trpc/client";
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
  const addPermission = trpc.permission.addPermissionToRootKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(true);
    },
    onSuccess: () => {
      toast.success("Permission added", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            removeRole.mutate({ rootKeyId, permissionName });
          },
        },
      });
    },
    onError: (error) => {
      toast.error(error.message);
    },
    onSettled: () => {
      router.refresh();
    },
  });
  const removeRole = trpc.permission.removePermissionFromRootKey.useMutation({
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
    onError: (error) => {
      toast.error(error.message);
    },
    onSettled: () => {
      router.refresh();
    },
  });

  return (
    <div className="flex items-center gap-8">
      <div className="w-1/3 ">
        <Tooltip>
          <TooltipTrigger className="flex items-center gap-2">
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
                      addPermission.mutate({ rootKeyId, permission: permissionName });
                    }
                  }
                }}
              />
            )}
            <Label className="text-xs text-content">{label}</Label>
          </TooltipTrigger>
          <TooltipContent className="flex items-center gap-2">
            <span className="font-mono text-sm font-medium">{permissionName}</span>
            <CopyButton value={permissionName} />
          </TooltipContent>
        </Tooltip>
      </div>

      <p className="w-2/3 text-xs text-content-subtle">{description}</p>
    </div>
  );
};
