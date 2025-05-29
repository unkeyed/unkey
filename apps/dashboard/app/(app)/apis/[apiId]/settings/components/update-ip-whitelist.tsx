"use client";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Workspace } from "@unkey/db";
import { ArrowUpRight, Lock, Shield } from "@unkey/icons";
import { Button, FormTextarea, InlineLink, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { Controller, useForm } from "react-hook-form";
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

  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting, errors, isDirty },
  } = useForm<z.infer<typeof formSchema>>({
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
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title={
          <div className="flex items-center justify-start gap-2.5">
            <span className="text-sm font-medium text-accent-12">IP Whitelist</span>
            <Badge />
          </div>
        }
        description={
          <div className="font-normal text-[13px]">
            Want to protect your API from unauthorized access? Upgrade to our{" "}
            <span className="font-bold">Enterprise plan</span> to enable IP whitelisting and
            restrict access to trusted sources.{" "}
            <InlineLink
              label="Learn more"
              href="https://www.unkey.com/docs/apis/features/whitelist#ip-whitelisting"
              target
              icon={<ArrowUpRight size="sm-thin" />}
            />
          </div>
        }
        border="both"
        contentWidth="w-full"
      >
        {isEnabled ? (
          <div className="flex flex-row justify-end items-start w-full gap-x-2">
            <input type="hidden" name="workspaceId" value={api.workspaceId} />
            <input type="hidden" name="apiId" value={api.id} />

            <Controller
              control={control}
              name="ipWhitelist"
              render={({ field }) => (
                <FormTextarea
                  {...field}
                  className="lg:w-[16rem]"
                  autoComplete="off"
                  placeholder={"127.0.0.1\n1.1.1.1"}
                  error={errors.ipWhitelist?.message}
                />
              )}
            />

            <Button
              size="lg"
              variant="primary"
              disabled={!isValid || isSubmitting || !isDirty}
              type="submit"
              loading={isSubmitting}
            >
              Save
            </Button>
          </div>
        ) : (
          <div className="flex flex-col justify-center items-end w-full h-full">
            <a target="_blank" rel="noreferrer" href="https://cal.com/james-r-perkins/sales">
              <Button type="button" size="lg" variant="primary" color="info">
                Upgrade to Enterprise
              </Button>
            </a>
          </div>
        )}
      </SettingCard>
    </form>
  );
};
