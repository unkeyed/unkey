"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
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
      toast.loading("Adding Permission");
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
      toast.loading("Removing Permission");
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
  );
};
