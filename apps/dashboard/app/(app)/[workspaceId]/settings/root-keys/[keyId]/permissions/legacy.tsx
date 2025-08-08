"use client";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { trpc } from "@/lib/trpc/client";
import type { Permission } from "@unkey/db";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  toast,
} from "@unkey/ui";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";

type Props = {
  permissions: Permission[];
  keyId: string;
};

export const Legacy: React.FC<Props> = ({ keyId, permissions }) => {
  const router = useRouter();
  const removeRole = trpc.rbac.removePermissionFromRootKey.useMutation({
    onSuccess: () => {
      toast.success("Role removed", {
        description: "Changes may take up to 60 seconds to take effect.",
      });
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
    onSettled: () => {},
  });

  /**
   * If they delete it without setting permissions first, they might break themselves in production.
   */
  const canSafelyDelete = permissions.filter((p) => p.id !== "*").length >= 1;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Legacy</CardTitle>
        <CardDescription>
          Existing keys were able to perform any action. Please remove this yourself at a convenient
          time to increase security.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Alert>
          <AlertDescription>
            Before you remove the legacy role, you need to add the correct permissions to the key,
            otherwise your key might not have sufficient permissions and you might break your
            application.
          </AlertDescription>
        </Alert>
      </CardContent>

      <CardFooter className="flex justify-end">
        <Button
          variant="destructive"
          disabled={!canSafelyDelete}
          onClick={() => {
            if (!canSafelyDelete) {
              toast.error(
                "You need to add at least one permissions before removing the legacy role. Otherwise the key will be useless.",
              );
              return;
            }
            removeRole.mutate({ rootKeyId: keyId, permissionName: "*" });
          }}
        >
          {removeRole.isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
          Remove Legacy Role
        </Button>
      </CardFooter>
    </Card>
  );
};
