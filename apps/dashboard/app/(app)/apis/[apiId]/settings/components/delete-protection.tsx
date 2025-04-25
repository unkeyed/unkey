"use client";
import { DialogContainer } from "@/components/dialog-container";
import { SettingCard } from "@/components/settings-card";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowUpRight, TriangleWarning2 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidate } from "../actions";
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
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
    },
  });

  const router = useRouter();
  const isValid = watch("name") === api.name;

  const updateDeleteProtection = trpc.api.updateDeleteProtection.useMutation({
    async onSuccess(_, { enabled }) {
      toast.message(
        `Delete protection for ${api.name} has been ${enabled ? "enabled" : "disabled"}`,
        {},
      );
      setOpen(false);
      await revalidate();
      reset();

      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
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
        <div className="flex items-center justify-start gap-2.5">
          <span className="text-sm font-medium text-accent-12">Delete Protection</span>{" "}
          <StatusBadge
            variant={api.deleteProtection ? "enabled" : "disabled"}
            text={api.deleteProtection ? "Enabled" : "Disabled"}
            icon={<TriangleWarning2 size="sm-thin" />}
          />
        </div>
      }
      description={
        api.deleteProtection ? (
          <div className="font-normal text-[13px] max-w-[380px]">
            Disabling this allows the API, along with all keys and data, to be deleted.
          </div>
        ) : (
          <div className="font-normal text-[13px] max-w-[380px]">
            Enabling this prevents the API from being deleted.
          </div>
        )
      }
      border="top"
      contentWidth="w-full lg:w-[320px]"
    >
      <div className="flex w-full gap-2 lg:items-center lg:justify-end">
        {api.deleteProtection ? (
          <Button
            type="button"
            variant="outline"
            color="warning"
            size="xlg"
            className="w-[24rem] lg:w-[12rem]"
            onClick={() => setOpen(true)}
          >
            Disable Delete Protection
          </Button>
        ) : (
          <Button
            type="button"
            variant="outline"
            color="success"
            size="xlg"
            className="w-[24rem] lg:w-[12rem]"
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
              form="delete-protection-form" // Connect to form ID
              variant="primary"
              color={api.deleteProtection ? "warning" : "success"}
              size="xlg"
              disabled={!isValid || updateDeleteProtection.isLoading || isSubmitting}
              loading={updateDeleteProtection.isLoading || isSubmitting}
              className="w-full"
            >
              {api.deleteProtection
                ? "Disable API Delete Protection"
                : "Enable API Delete Protection"}
            </Button>
            <div className="font-normal text-[12px] text-gray-9 text-center">
              This setting can be {!api.deleteProtection ? "disabled" : "re-enabled"} at any time
            </div>
          </div>
        }
      >
        <div className="flex flex-col gap-4">
          <p className="text-gray-11 text-[13px]">
            <span className="font-medium">Important: </span>
            {!api.deleteProtection
              ? "Enabling this prevents the API from being deleted. This setting can be disabled at any time."
              : "Disabling this allows API deletion. This setting can be re-enabled at any time."}
            <a href="https://www.unkey.com/docs/security/delete-protection">
              <Button
                type="button"
                variant="ghost"
                className="py-3 gap-1 font-medium text-accent-12 leading-4 text-[13px]"
              >
                Learn more <ArrowUpRight size="lg-thin" className="text-accent-9" />
              </Button>
            </a>
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
