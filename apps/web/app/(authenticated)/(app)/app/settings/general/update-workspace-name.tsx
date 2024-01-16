"use client";
import { Button } from "@/components/ui/button";
import React from "react";
import { useFormStatus } from "react-dom";

import { Loading } from "@/components/dashboard/loading";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { useUser } from "@clerk/nextjs";
import { updateWorkspaceName } from "./actions";
export const dynamic = "force-dynamic";
type Props = {
  workspace: {
    id: string;
    tenantId: string;
    name: string;
  };
};

export const UpdateWorkspaceName: React.FC<Props> = ({ workspace }) => {
  const { user } = useUser();
  const { pending } = useFormStatus();

  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateWorkspaceName(formData);
        if (res.error) {
          toast.error(res.error.message);
          return;
        }

        toast.success("Workspace name updated");
        user?.reload();
      }}
    >
      <Card>
        <CardHeader>
          <CardTitle>Workspace Name</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            <input type="hidden" name="workspaceId" value={workspace.id} />
            <label className="sr-only hidden">Name</label>
            <Input name="name" className="max-w-sm" defaultValue={workspace.name} />
            <p className="text-content-subtle text-xs">What should your workspace be called?</p>
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button variant={pending ? "disabled" : "primary"} type="submit" disabled={pending}>
            {pending ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
