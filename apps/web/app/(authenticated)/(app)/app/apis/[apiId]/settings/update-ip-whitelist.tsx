"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import React, { useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { FormField } from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Workspace } from "@unkey/db";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  ipWhitelist: z.string(),
  apiId: z.string(),
  workspaceId: z.string(),
});

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
  const [isLoading, setLoading] = useState(false);
  const isEnabled = workspace.plan === "enterprise";
  const updateIps = trpc.api.updateIpWhitelist.useMutation();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      ipWhitelist: api.ipWhitelist ?? "",
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    try {
      setLoading(true);
      await updateIps.mutate({
        ips: values.ipWhitelist,
        apiId: values.apiId,
        workspaceId: values.workspaceId,
      });
      toast({
        title: "Success",
        description: "Your ip whitelist has been updated!",
      });
      router.refresh();
    } catch (err) {
      toast({
        title: "Error",
        description: (err as Error).message,
        variant: "alert",
      });
    } finally {
      setLoading(false);
    }
  }
  const router = useRouter();

  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
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
              <label className="hidden sr-only">Name</label>
              <FormField
                control={form.control}
                name="ipWhitelist"
                render={({ field }) => (
                  <Textarea
                    className="max-w-sm"
                    {...field}
                    defaultValue={field.value}
                    autoComplete="off"
                    placeholder={`127.0.0.1
1.1.1.1`}
                  />
                )}
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
            variant={!isEnabled || isLoading ? "disabled" : "primary"}
            type="submit"
            disabled={isLoading}
          >
            {isLoading ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
