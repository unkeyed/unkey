"use client";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";

type Props = {
  permissions: string[];
  keyId: string;
};

export const Legacy: React.FC<Props> = ({ keyId, permissions }) => {
  const router = useRouter();
  const removeRole = trpc.permission.removeRoleFromRootKey.useMutation({
    onSuccess: () => {
      toast.success("Role removed", {
        description: "Changes may take up to 60 seconds to take effect.",
      });
      router.refresh();
    },
    onError: (error) => {
      toast.error(error.message);
    },
    onSettled: () => {},
  });

  /**
   * If they delete it without setting permissions first, they might break themselves in production.
   */
  const canSafelyDelete = permissions.filter((p) => p !== "*").length >= 1;

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
          variant={canSafelyDelete ? "alert" : "disabled"}
          disabled={!canSafelyDelete}
          onClick={() => {
            if (!canSafelyDelete) {
              toast.error(
                "You need to add at least one permissions before removing the legacy role. Otherwise the key will be useless.",
              );
              return;
            }
            removeRole.mutate({ rootKeyId: keyId, role: "*" });
          }}
        >
          {removeRole.isLoading ? <Loader2 className="animate-spin w-4 h-4" /> : null}
          Remove Legacy Role
        </Button>
      </CardFooter>
    </Card>
  );
};
