"use client";
import { SettingCard } from "@/components/settings-card";
import { FormField } from "@/components/ui/form";
import { AnimatedShinyText } from "@/components/ui/shiny-text";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Workspace } from "@unkey/db";
import { AdjustContrast3, ArrowUpRight, Lock, Shield } from "@unkey/icons";
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
        className="mt-[20px] py-[19px]"
        title={
          <div className=" flex items-center justify-start gap-2.5">
            <AdjustContrast3 size="xl-medium" className="h-full text-brand-10" />
            <span className="text-sm font-medium text-accent-12">IP Whitelist</span>
            <Badge />
          </div>
        }
        description={
          <div className="font-normal text-[13px]">
            Want to protect your API keys from unauthorized access? <br />
            Upgrade to our <span className="font-bold">Enterprise plan</span> to enable IP
            whitelisting <br /> and restrict access to trusted sources.
          </div>
        }
        border="both"
      >
        {isEnabled ? (
          <div className="flex flex-row items-start justify-end w-full gap-2">
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
          <div className="flex items-center justify-end w-full gap-4">
            <a
              href="https://cal.com/james-r-perkins/sales"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-center px-1 border rounded-lg border-grayA-4 h-9"
            >
              <AnimatedShinyText
                shimmerWidth={50}
                className="py-1 px-[9px] rounded-lg text-[13px] leading-6 font-medium"
              >
                Upgrade to Enterprise...
              </AnimatedShinyText>
            </a>
            <a
              href="https://www.unkey.com/docs/apis/features/whitelist#ip-whitelisting"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-end gap-1 font-medium text-accent-12 leading-4 text-[13px]"
            >
              Learn more <ArrowUpRight size="lg-thin" className="text-accent-9" />
            </a>
          </div>
        )}
      </SettingCard>
    </form>
  );
};
