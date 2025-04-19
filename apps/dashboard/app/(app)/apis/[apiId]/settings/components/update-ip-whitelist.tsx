"use client";
import { SettingCard } from "@/components/settings-card";
import { FormField } from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Workspace } from "@unkey/db";
import { ArrowUpRight, Lock, Shield } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { StatusBadge } from "./status-badge";

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
  const Badge = () =>
    isEnabled ? (
      <StatusBadge variant="enabled" text="Enabled" icon={<Shield size="sm-thin" />} />
    ) : (
      <StatusBadge variant="locked" text="Locked" icon={<Lock size="sm-thin" />} />
    );
  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <SettingCard
        className="mt-[20px] py-[19px] w-full"
        title={
          <div className=" flex items-center justify-start gap-2.5">
            <span className="text-sm font-medium text-accent-12">IP Whitelist</span>
            <Badge />
          </div>
        }
        description={
          <div className="font-normal text-[13px] max-w-[380px]">
            Want to protect your API keys from unauthorized access? <br />
            Upgrade to our <span className="font-bold">Enterprise plan</span> to enable IP
            whitelisting and restrict access to trusted sources.
          </div>
        }
        border="both"
        contentWidth="w-full lg:w-[320px]"
      >
        {isEnabled ? (
          <div className="flex flex-row justify-items-stretch items-center w-full gap-x-2">
            <input type="hidden" name="workspaceId" value={api.workspaceId} />
            <input type="hidden" name="apiId" value={api.id} />
            <label htmlFor="ipWhitelist" className="hidden sr-only">
              IP Whitelist
            </label>
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
            <Button
              size="lg"
              className="rounded-lg px-2.5"
              disabled={
                !form.formState.isValid ||
                form.formState.isSubmitting ||
                api.ipWhitelist === form.watch("ipWhitelist")
              }
              type="submit"
              loading={form.formState.isSubmitting}
            >
              Save
            </Button>
          </div>
        ) : (
          <div className="flex w-full gap-4 lg:justify-end lg:items-center ">
            <a target="_blank" rel="noreferrer" href="https://cal.com/james-r-perkins/sales">
              <Button
                type="button"
                variant="primary"
                className="flex items-center justify-end gap-1 font-medium text-info-11 bg-info-4 leading-4 text-[13px] px-3 h-9 rounded-lg border border-info-5 hover:bg-info-3 w-[16rem] lg:w-[12rem]"
              >
                Upgrade to Enterprise
              </Button>
            </a>
            <a
              href="https://www.unkey.com/docs/apis/features/whitelist#ip-whitelisting"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-end"
            >
              <Button
                type="button"
                variant="ghost"
                className="py-3 gap-1 font-medium text-accent-12 leading-4 text-[13px]"
              >
                Learn more <ArrowUpRight size="lg-thin" className="text-accent-9" />
              </Button>
            </a>
          </div>
        )}
      </SettingCard>
    </form>
  );
};
