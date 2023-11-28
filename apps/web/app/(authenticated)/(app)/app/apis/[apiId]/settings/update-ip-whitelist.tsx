"use client";
import { Button } from "@/components/ui/button";
import React from "react";
import { useFormStatus } from "react-dom";

import { Loading } from "@/components/dashboard/loading";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { updateIpWhitelist } from "./actions";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { Workspace } from "@unkey/db";
import Link from "next/link";
type Props = {
  workspace: {
    plan: Workspace["plan"];
  };
  api: {
    id: string;
    workspaceId: string;
    name: string;
    ipWhitelist: string | null;
  };
};

export const UpdateIpWhitelist: React.FC<Props> = ({ api, workspace }) => {
  const { toast } = useToast();
  const { pending } = useFormStatus();

  const isEnabled = workspace.plan === "enterprise";

  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateIpWhitelist(formData);
        if (res.error) {
          toast({
            title: "Error",
            description: res.error.message,
            variant: "alert",
          });
          return;
        }
        toast({
          title: "Success",
          description: "Api name updated",
        });
      }}
    >
      <Card>
        <CardHeader className={cn({ "opacity-40": !isEnabled })}>
          <CardTitle>IP Whitelist</CardTitle>
          <CardDescription>
            Protect your keys from being verified by unauthorized sources. Enter your IP addresses
            either comma or newline separated.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {workspace.plan === "enterprise" ? (
            <div className="flex flex-col space-y-2">
              <input type="hidden" name="workspaceId" value={api.workspaceId} />
              <input type="hidden" name="apiId" value={api.id} />
              <label className="sr-only hidden">Name</label>
              <Textarea
                name="ipWhitelist"
                className="max-w-sm"
                defaultValue={api.ipWhitelist ?? ""}
                autoComplete="off"
                placeholder={`127.0.0.1
1.1.1.1`}
              />
            </div>
          ) : (
            <Alert className="flex items-center justify-between opacity-100">
              <div>
                <AlertTitle>Enterprise Feature</AlertTitle>
                <AlertDescription>
                  IP whitelists are only available on the enterprise plan.
                </AlertDescription>
              </div>
              <Link href="mailto:support@unkey.dev">
                <Button>Upgrade</Button>
              </Link>
            </Alert>
          )}
        </CardContent>
        <CardFooter className={cn("justify-end", { "opacity-30 ": !isEnabled })}>
          <Button
            variant={!isEnabled || pending ? "disabled" : "primary"}
            type="submit"
            disabled={pending}
          >
            {pending ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
