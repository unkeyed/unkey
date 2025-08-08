"use client";
import { trpc } from "@/lib/trpc/client";
import { ArrowUpRight, TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, InlineLink, Input, SettingCard } from "@unkey/ui";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { createApiFormConfig, createMutationHandlers } from "./key-settings-form-helper";
import { StatusBadge } from "./status-badge";

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
    deleteProtection: boolean | null;
  };
};

export const DeleteProtection: React.FC<Props> = ({ api }) => {
  const { onDeleteProtectionSuccess, onError } = createMutationHandlers();
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the API name"),
  });

  type FormValues = z.infer<typeof formSchema>;

  const {
    register,
    handleSubmit,
    watch,
    reset,
    formState: { isSubmitting },
  } = useForm<FormValues>({
    ...createApiFormConfig(formSchema),
    mode: "onChange",
    defaultValues: {
      name: "",
    },
  });

  const isValid = watch("name") === api.name;

  const updateDeleteProtection = trpc.api.updateDeleteProtection.useMutation({
    async onSuccess(_, { enabled }) {
      onDeleteProtectionSuccess(api.name, enabled)();
      setOpen(false);
      reset();
    },
    onError,
  });

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    updateDeleteProtection.mutate({
      apiId: api.id,
      enabled: !api.deleteProtection,
    });
  }

  return (
    <SettingCard
      title={
        <div className="inline-flex gap-2">
          <span>Delete Protection</span>{" "}
          <StatusBadge
            variant={api.deleteProtection ? "enabled" : "disabled"}
            text={api.deleteProtection ? "Enabled" : "Disabled"}
            icon={<TriangleWarning2 size="sm-thin" />}
          />
        </div>
      }
      description={
        api.deleteProtection ? (
          <p>Disabling this allows the API, along with all keys and data, to be deleted.</p>
        ) : (
          <p>Enabling this prevents the API from being deleted.</p>
        )
      }
      border="top"
      className="border-b-1"
    >
      <div className="flex w-full gap-2 lg:items-center justify-end">
        {api.deleteProtection ? (
          <Button
            type="button"
            variant="outline"
            color="warning"
            className="h-full px-3.5 "
            size="xlg"
            onClick={() => setOpen(true)}
          >
            Disable Delete Protection
          </Button>
        ) : (
          <Button
            type="button"
            className="h-full px-3.5 "
            variant="outline"
            color="success"
            size="xlg"
            onClick={() => setOpen(true)}
          >
            Enable Delete Protection
          </Button>
        )}
      </div>
      <DialogContainer
        isOpen={open}
        onOpenChange={setOpen}
        title={`${api.deleteProtection ? "Disable" : "Enable"} API Delete Protection`}
        footer={
          <div className="flex flex-col gap-2 items-center justify-center w-full">
            <Button
              type="submit"
              form="delete-protection-form"
              variant="primary"
              color={api.deleteProtection ? "warning" : "success"}
              size="xlg"
              disabled={!isValid || isSubmitting}
              loading={isSubmitting}
              className="w-full"
            >
              {api.deleteProtection
                ? "Disable API Delete Protection"
                : "Enable API Delete Protection"}
            </Button>
            <div className="font-normal text-[12px] text-gray-9 text-center">
              This setting can be {api.deleteProtection ? "disabled" : "enabled"} at any time
            </div>
          </div>
        }
      >
        <div className="flex flex-col gap-4">
          <p className="text-gray-11 text-[13px]">
            <span className="font-medium">Important: </span>
            {api.deleteProtection
              ? "Disabling this allows API deletion. This setting can be re-enabled at any time. "
              : "Enabling this prevents the API from being deleted. This setting can be disabled at any time. "}
            <InlineLink
              label="Learn more"
              target="_blank"
              rel="noopener noreferrer"
              href="https://www.unkey.com/docs/security/delete-protection"
              icon={<ArrowUpRight size="sm-thin" />}
            />
          </p>
          <form id="delete-protection-form" onSubmit={handleSubmit(onSubmit)}>
            <div className="space-y-1">
              <p className="text-gray-11 text-[13px]">
                Type <span className="text-gray-12 font-medium">{api.name}</span> to confirm
              </p>
              <Input {...register("name")} placeholder={`Enter "${api.name}" to confirm`} />
            </div>
          </form>
        </div>
      </DialogContainer>
    </SettingCard>
  );
};
