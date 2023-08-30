"use client";
import { Button } from "@/components/ui/button";
import React from "react";
import { experimental_useFormStatus as useFormStatus } from "react-dom";

import { Loading } from "@/components/dashboard/loading";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useToast } from "@/components/ui/use-toast";
import { useUser } from "@clerk/nextjs";
import { updateWorkspaceName } from "./actions";
type Props = {
  workspace: {
    id: string;
    tenantId: string;
    name: string;
  };
};

export const UpdateWorkspaceName: React.FC<Props> = ({ workspace }) => {
  const { toast } = useToast();
  const { user } = useUser();
  const { pending } = useFormStatus();

  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateWorkspaceName(formData);
        if (res.error) {
          toast({
            title: "Error",
            description: res.error,
            variant: "alert",
          });
          return;
        }
        toast({
          title: "Success",
          description: "Workspace name updated",
        });

        user?.reload();
      }}
    >
      <div className="flex flex-col space-y-2">
        <input type="hidden" name="workspaceId" value={workspace.id} />
        <Label>Name</Label>
        <Input name="name" defaultValue={workspace.name} />
        <p className="text-xs text-content-subtle">What should your workspace be called?</p>
      </div>

      <Button variant="primary" type="submit" size="sm" disabled={pending}>
        {pending ? <Loading /> : "Save"}
      </Button>
    </form>
  );
};
