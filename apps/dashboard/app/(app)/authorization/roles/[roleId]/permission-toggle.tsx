"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "@/components/ui/toaster";
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
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
    onSettled: () => {
      router.refresh();
    },
  });
  const disconnect = trpc.rbac.disconnectPermissionFromRole.useMutation({
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
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
    onSettled: () => {
      router.refresh();
    },
  });
  if (connect.isLoading || disconnect.isLoading) {
    return <Loader2 className="w-4 h-4 animate-spin" />;
  }
  return (
    <Checkbox
      className="size-[14px]"
      checked={optimisticChecked}
      onClick={() => {
        if (optimisticChecked) {
          disconnect.mutate({ roleId, permissionId });
        } else {
          connect.mutate({ roleId, permissionId });
        }
      }}
    />
  );
};
