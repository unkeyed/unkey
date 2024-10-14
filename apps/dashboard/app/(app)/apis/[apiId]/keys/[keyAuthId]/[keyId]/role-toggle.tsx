"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { CircleCheck, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
type Props = {
  keyId: string;
  roleId: string;
  checked: boolean;
};

export const RoleToggle: React.FC<Props> = ({ roleId, keyId, checked }) => {
  const router = useRouter();

  const [optimisticChecked, setOptimisticChecked] = useState(checked);
  const connect = trpc.rbac.connectRoleToKey.useMutation({
    onSettled: () => {
      router.refresh();
    },
  });
  const disconnect = trpc.rbac.disconnectRoleFromKey.useMutation({
    onSettled: () => {
      router.refresh();
    },
  });

  const handleConnect = async ({ roleId, keyId }: { roleId: string; keyId: string }) => {
    setOptimisticChecked(true);
    toast.promise(connect.mutateAsync({ roleId, keyId }), {
      loading: "Adding Role ...",
      success: () => (
        <div className="flex items-center gap-2">
          <CircleCheck className="w-4 h-4 text-gray-300" />

          <div className="max-w-[250px]">
            <h2 className="font-semibold">Role added</h2>
            <p>Changes may take up to 60 seconds to take effect.</p>
          </div>

          <div>
            <button
              onClick={() => disconnect.mutate({ roleId, keyId })}
              className="bg-[#2c2c2c] text-white px-[4px] py-[2px] rounded hover:bg-[#3c3c3c] transition-colors"
            >
              Undo
            </button>
          </div>
        </div>
      ),
      error: (error) => `${error.message || "An error occurred while adding the Role ."}`,
    });
  };

  const handleDisconnect = async ({ roleId, keyId }: { roleId: string; keyId: string }) => {
    setOptimisticChecked(false);

    toast.promise(disconnect.mutateAsync({ roleId, keyId }), {
      loading: "Removing Role...",
      success: () => (
        <div className="flex items-center gap-2">
          <CircleCheck className="w-4 h-4 text-gray-300" />

          <div className="max-w-[250px]">
            <h2 className="font-semibold">Role removed</h2>
            <p>Changes may take up to 60 seconds to take effect.</p>
          </div>

          <div>
            <button
              onClick={() => connect.mutate({ roleId, keyId })}
              className="bg-[#2c2c2c] text-white px-[4px] py-[2px] rounded hover:bg-[#3c3c3c] transition-colors"
            >
              Undo
            </button>
          </div>
        </div>
      ),
      error: (error) => `${error.message || "An error occurred while removing the role."}`,
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
          handleDisconnect({ roleId, keyId });
        } else {
          handleConnect({ roleId, keyId });
        }
      }}
    />
  );
};
