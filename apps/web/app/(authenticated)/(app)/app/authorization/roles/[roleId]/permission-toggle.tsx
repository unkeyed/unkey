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
  permissionId: string;
  roleId: string;
  checked: boolean;
};

export const PermissionToggle: React.FC<Props> = ({ roleId, permissionId, checked }) => {
  const router = useRouter();

  const [optimisticChecked, setOptimisticChecked] = useState(checked);
  const connect = trpc.rbac.connectPermissionToRole.useMutation({
    onMutate: () => {
      setOptimisticChecked(true);
    },
    onSuccess: () => {
      toast.success("Permission added", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            disconnect.mutate({ roleId, permissionId });
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
  const disconnect = trpc.rbac.disconnectPermissionToRole.useMutation({
    onMutate: () => {
      setOptimisticChecked(false);
    },
    onSuccess: () => {
      toast.success("Permission removed", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            connect.mutate({ roleId, permissionId });
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
    <div>
      {connect.isLoading || disconnect.isLoading ? (
        <Loader2 className="w-4 h-4 animate-spin" />
      ) : (
        <Checkbox
          disabled={connect.isLoading || disconnect.isLoading}
          checked={optimisticChecked}
          onClick={() => {
            if (checked) {
              disconnect.mutate({ roleId, permissionId });
            } else {
              connect.mutate({ roleId, permissionId });
            }
          }}
        />
      )}
    </div>
  );
};
