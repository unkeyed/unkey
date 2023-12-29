"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { trpc } from "@/lib/trpc/client";
import { useRouter } from "next/navigation";
import { useState } from "react";
type Props = {
  rootKeyId: string;
  role: string;
  label: string;
  description: string;
  checked: boolean;
};

export const Permission: React.FC<Props> = ({ rootKeyId, role, label, checked, description }) => {
  const router = useRouter();

  const [optimisticChecked, setOptimisticChecked] = useState(checked);
  const addRole = trpc.permission.addRoleToRootKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(true);
    },
    onSuccess: () => {
      toast.success("Role added", {
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
            <Checkbox
              disabled={addRole.isLoading || removeRole.isLoading}
              checked={optimisticChecked}
              onClick={() => {
                if (checked) {
                  removeRole.mutate({ rootKeyId, role });
                } else {
                  addRole.mutate({ rootKeyId, role });
                }
              }}
            />

            <Label className="text-xs font-mono text-content">{label}</Label>
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
