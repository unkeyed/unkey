"use client";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowUpRight, Shield } from "@unkey/icons";
import { Button, FormTextarea, InlineLink, SettingCard } from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { createApiFormConfig, createMutationHandlers } from "./key-settings-form-helper";
import { StatusBadge } from "./status-badge";

const formSchema = z.object({
  ipWhitelist: z.string(),
  apiId: z.string(),
  workspaceId: z.string(),
});

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
    ipWhitelist: string | null;
  };
};

export const UpdateIpWhitelist: React.FC<Props> = ({ api }) => {
  const { onUpdateSuccess, onError } = createMutationHandlers();

  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting, errors, isDirty },
  } = useForm<z.infer<typeof formSchema>>({
    ...createApiFormConfig(formSchema),
    resolver: zodResolver(formSchema),
    defaultValues: {
      ipWhitelist: api.ipWhitelist ?? "",
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  const updateIps = trpc.api.updateIpWhitelist.useMutation({
    onSuccess: onUpdateSuccess("IP whitelist updated successfully"),
    onError,
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await updateIps.mutateAsync(values);
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title={
          <div className="flex items-center justify-start gap-2.5">
            <span className="text-sm font-medium text-accent-12">IP Whitelist</span>
            <StatusBadge variant="enabled" text="Enabled" icon={<Shield iconSize="sm-thin" />} />
          </div>
        }
        description={
          <div className="font-normal text-[13px]">
            Restrict access to this API to a set of trusted IP addresses. Leave empty to allow
            access from any IP.{" "}
            <InlineLink
              label="Learn more"
              target="_blank"
              rel="noopener noreferrer"
              href="https://www.unkey.com/docs/apis/features/whitelist#ip-whitelisting"
              icon={<ArrowUpRight iconSize="sm-thin" />}
            />
          </div>
        }
        contentWidth="w-full"
      >
        <div className="flex flex-row justify-end items-start w-full gap-x-2">
          <input type="hidden" name="workspaceId" value={api.workspaceId} />
          <input type="hidden" name="apiId" value={api.id} />

          <Controller
            control={control}
            name="ipWhitelist"
            render={({ field }) => (
              <FormTextarea
                {...field}
                className="lg:w-64"
                autoComplete="off"
                placeholder={"127.0.0.1\n1.1.1.1"}
                error={errors.ipWhitelist?.message}
              />
            )}
          />

          <Button
            size="lg"
            variant="primary"
            className="w-fit px-3.5"
            disabled={!isValid || isSubmitting || !isDirty}
            type="submit"
            loading={isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};
