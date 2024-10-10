"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
type Props = {
  keyId: string;
  roleId: string;
  checked: boolean;
};

export const RoleToggle: React.FC<Props> = ({ roleId, keyId, checked }) => {
  const router = useRouter();
  const loadingToastId = useRef<string | number | null>(null);

  const [optimisticChecked, setOptimisticChecked] = useState(checked);
  const connect = trpc.rbac.connectRoleToKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(true);

      const id = toast.loading("Adding Role");
      loadingToastId.current = id;
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
      toast.dismiss(loadingToastId.current!);
      loadingToastId.current = null
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
      toast.dismiss(loadingToastId.current!);
      loadingToastId.current = null
    },
    onSettled: () => {
      router.refresh();
    },
  });
  const disconnect = trpc.rbac.disconnectRoleFromKey.useMutation({
    onMutate: () => {
      setOptimisticChecked(false);
      const id = toast.loading("Removing role");
      loadingToastId.current = id
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
      toast.dismiss(loadingToastId.current!);
      loadingToastId.current = null
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
      toast.dismiss(loadingToastId.current!);
      loadingToastId.current = null
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
