"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Lock } from "@unkey/icons";
import { Button, DialogContainer, Input, SettingsZoneRow } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { createApiFormConfig, createMutationHandlers } from "./key-settings-form-helper";
import { StatusBadge } from "./status-badge";

type Props = {
  keys: number;
  api: {
    id: string;
    name: string;
    deleteProtection: boolean | null;
  };
};

export const DeleteApi: React.FC<Props> = ({ api, keys }) => {
  const workspace = useWorkspaceNavigation();
  const { onDeleteSuccess, onError } = createMutationHandlers();
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const intent =
    keys > 0
      ? `delete this keyspace and ${keys} key${keys > 1 ? "s" : ""}`
      : "delete this keyspace";

  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the keyspace name"),
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  type FormValues = z.infer<typeof formSchema>;

  const {
    register,
    handleSubmit,
    watch,
    formState: { isSubmitting },
  } = useForm<FormValues>({
    ...createApiFormConfig(formSchema),
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      name: "",
      intent: "",
    },
  });

  const isValid = watch("name") === api.name && watch("intent") === intent;

  const deleteApi = trpc.api.delete.useMutation({
    async onSuccess() {
      onDeleteSuccess(keys)();
      router.push(routes.apis.list({ workspaceSlug: workspace.slug }));
    },
    onError,
  });

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    deleteApi.mutate({ apiId: api.id });
  }

  return (
    <>
      <SettingsZoneRow
        title={
          <div className="inline-flex gap-2">
            <span>Delete Keyspace</span>
            {api.deleteProtection && (
              <StatusBadge variant="locked" text="Locked" icon={<Lock iconSize="sm-thin" />} />
            )}
          </div>
        }
        description={
          api.deleteProtection
            ? "Permanently deletes this keyspace, including all keys and data. This action is locked by Delete Protection."
            : "Permanently deletes this keyspace, including all keys and data. This action cannot be undone."
        }
        action={{
          label: "Delete Keyspace",
          onClick: () => setOpen(true),
          disabled: api.deleteProtection === true,
        }}
      />
      <DialogContainer
        isOpen={open}
        onOpenChange={setOpen}
        title="Delete Keyspace"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="delete-api-form"
              variant="primary"
              color="danger"
              size="xlg"
              disabled={api.deleteProtection || !isValid || isSubmitting}
              loading={isSubmitting}
              className="w-full"
            >
              Delete Keyspace
            </Button>
            <div className="text-gray-9 text-xs">
              This action cannot be undone – proceed with caution
            </div>
          </div>
        }
      >
        <p className="text-gray-11 text-[13px]">
          <span className="font-medium">Warning: </span>
          Deleting this keyspace will delete all keys and data associated with it. This action
          cannot be undone. Any tracking, enforcement, and historical insights tied to this keyspace
          will be permanently lost.
        </p>
        <form id="delete-api-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-1">
            <p className="text-gray-11 text-[13px]">
              Type <span className="text-gray-12 font-medium">{api.name}</span> to confirm
            </p>
            <Input {...register("name")} placeholder={`Enter "${api.name}" to confirm`} />
          </div>
          <div className="flex flex-col gap-1 mt-6">
            <p className="text-gray-11 text-[13px]">
              To verify, type <span className="text-gray-12 font-medium">{intent}</span> to confirm
            </p>
            <Input {...register("intent")} placeholder={`Enter "${intent}" to confirm`} />
          </div>
        </form>
      </DialogContainer>
    </>
  );
};
