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
  role: string;
  label: string;
  description: string;
  checked: boolean;
  preventEnabling?: boolean;
  preventDisabling?: boolean;
};

export const Permission: React.FC<Props> = ({
  rootKeyId,
  role,
  label,
  checked,
  description,
  preventEnabling,
  preventDisabling,
}) => {
  const router = useRouter();

  const [optimisticChecked, setOptimisticChecked] = useState(checked);
  const addRole = trpc.permission.addRoleToRootKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(true);
    },
    onSuccess: () => {
      toast.success("Role added", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            removeRole.mutate({ rootKeyId, role });
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
  const removeRole = trpc.permission.removeRoleFromRootKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(false);
    },
    onSuccess: () => {
      toast.success("Role removed", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            addRole.mutate({ rootKeyId, role });
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
            {addRole.isLoading || removeRole.isLoading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Checkbox
                disabled={
                  addRole.isLoading ||
                  removeRole.isLoading ||
                  (preventEnabling && !checked) ||
                  (preventDisabling && checked)
                }
                checked={optimisticChecked}
                onClick={() => {
                  if (checked) {
                    if (!preventDisabling) {
                      removeRole.mutate({ rootKeyId, role });
                    }
                  } else {
                    if (!preventEnabling) {
                      addRole.mutate({ rootKeyId, role });
                    }
                  }
                }}
              />
            )}
            <Label className="text-xs text-content">{label}</Label>
          </TooltipTrigger>
          <TooltipContent className="flex items-center gap-2">
            <span className="font-mono font-medium text-sm">{role}</span>
            <CopyButton value={role} />
          </TooltipContent>
        </Tooltip>
      </div>

      <p className="text-xs w-2/3 text-content-subtle">{description}</p>
    </div>
  );
};
