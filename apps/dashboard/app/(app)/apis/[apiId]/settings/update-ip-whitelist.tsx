"use client";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FormField } from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Workspace } from "@unkey/db";
import { Button } from "@unkey/ui";
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
    features: Workspace["features"];
  };
  api: {
    id: string;
    workspaceId: string;
    name: string;
    ipWhitelist: string | null;
  };
};

export const UpdateIpWhitelist: React.FC<Props> = ({ api, workspace }) => {
  const router = useRouter();
  const isEnabled = workspace.features.ipWhitelist;

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      ipWhitelist: api.ipWhitelist ?? "",
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  const updateIps = trpc.api.updateIpWhitelist.useMutation({
    onSuccess() {
      toast.success("Your ip whitelist has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await updateIps.mutateAsync(values);
  }

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
          {isEnabled ? (
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
                    autoComplete="off"
                    placeholder={`127.0.0.1
1.1.1.1`}
                  />
                )}
              />
            </div>
          ) : (
            <Alert className="flex items-center justify-between opacity-100 border-border">
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
            variant="primary"
            disabled={!form.formState.isValid || form.formState.isSubmitting}
            type="submit"
            loading={form.formState.isSubmitting}
          >
            Save
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
