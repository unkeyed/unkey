"use client";;
import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "@/components/ui/toaster";
import { useTRPC } from "@/lib/trpc/client";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
type Props = {
  keyId: string;
  roleId: string;
  checked: boolean;
};

export const RoleToggle: React.FC<Props> = ({ roleId, keyId, checked }) => {
  const trpc = useTRPC();
  const router = useRouter();

  const [optimisticChecked, setOptimisticChecked] = useState(checked);
  const connect = useMutation(trpc.rbac.connectRoleToKey.mutationOptions({
    onMutate: () => {
      setOptimisticChecked(true);
      toast.loading("Adding Role");
    },
    onSuccess: () => {
      toast.success("Role added", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            disconnect.mutate({ roleId, keyId });
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
  }));
  const disconnect = useMutation(trpc.rbac.disconnectRoleFromKey.mutationOptions({
    onMutate: () => {
      setOptimisticChecked(false);
      toast.loading("Removing role");
    },
    onSuccess: () => {
      toast.success("Role removed", {
        description: "Changes may take up to 60 seconds to take effect.",
        cancel: {
          label: "Undo",
          onClick: () => {
            connect.mutate({ roleId, keyId });
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
  }));
  if (connect.isPending || disconnect.isPending) {
    return <Loader2 className="w-4 h-4 animate-spin" />;
  }
  return (
    <Checkbox
      checked={optimisticChecked}
      onClick={() => {
        if (optimisticChecked) {
          disconnect.mutate({ roleId, keyId });
        } else {
          connect.mutate({ roleId, keyId });
        }
      }}
    />
  );
};
