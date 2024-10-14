"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { CircleCheck, Loader2 } from "lucide-react";
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
    onSettled: () => {
      router.refresh();
    },
  });

  const disconnect = trpc.rbac.disconnectPermissionFromRole.useMutation({
    onSettled: () => {
      router.refresh();
    },
  });

  const handleConnect = async (roleId: string, permissionId: string) => {
    setOptimisticChecked(true);
    toast.promise(connect.mutateAsync({ roleId, permissionId }), {
      loading: "Adding Permission ...",
      success: () => (
        <div className="flex items-center gap-2">
          <CircleCheck className="w-4 h-4 text-gray-300" />

          <div className="max-w-[250px]">
            <h2 className="font-semibold">Permission added</h2>
            <p>Changes may take up to 60 seconds to take effect.</p>
          </div>

          <div>
            <button
              onClick={() => disconnect.mutate({ roleId, permissionId })}
              className="bg-[#2c2c2c] text-white px-[4px] py-[2px] rounded hover:bg-[#3c3c3c] transition-colors"
            >
              Undo
            </button>
          </div>
        </div>
      ),
      error: (error) => `${error.message || "An error occurred while adding the permission."}`,
    });
  };

  const handleDisconnect = async (roleId: string, permissionId: string) => {
    setOptimisticChecked(false);
    toast.promise(disconnect.mutateAsync({ roleId, permissionId }), {
      loading: "Removing Permission ...",
      success: () => (
        <div className="flex items-center gap-2">
          <CircleCheck className="w-4 h-4 text-gray-300" />

          <div className="max-w-[250px]">
            <h2 className="font-semibold">Permission removed</h2>
            <p>Changes may take up to 60 seconds to take effect.</p>
          </div>

          <div>
            <button
              onClick={() => connect.mutate({ roleId, permissionId })}
              className="bg-[#2c2c2c] text-white px-[4px] py-[2px] rounded hover:bg-[#3c3c3c] transition-colors"
            >
              Undo
            </button>
          </div>
        </div>
      ),
      error: (error) => `${error.message || "An error occurred while removing the permission."}`,
    });
  };

  if (connect.isLoading || disconnect.isLoading) {
    return <Loader2 className="w-4 h-4 animate-spin" />;
  }
  return (
    <Checkbox
      checked={optimisticChecked}
      onClick={() => {
        if (optimisticChecked) {
          handleDisconnect(roleId, permissionId);
        } else {
          handleConnect(roleId, permissionId);
        }
      }}
    />
  );
};
