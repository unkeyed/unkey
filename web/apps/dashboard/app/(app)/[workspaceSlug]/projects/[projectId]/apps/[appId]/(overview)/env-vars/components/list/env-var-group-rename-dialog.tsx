"use client";

import { collection } from "@/lib/collections";
import { envVarKeySchema } from "@/lib/schemas/env-var";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useForm } from "react-hook-form";
import { z } from "zod";
import type { EnvVarItem } from "./env-var-item-row";

const renameSchema = z.object({
  key: envVarKeySchema,
});

type RenameFormValues = z.infer<typeof renameSchema>;

type EnvVarGroupRenameDialogProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  groupKey: string;
  items: EnvVarItem[];
};

export function EnvVarGroupRenameDialog({
  isOpen,
  onOpenChange,
  groupKey,
  items,
}: EnvVarGroupRenameDialogProps) {
  const renameMutation = trpc.deploy.envVar.rename.useMutation();

  const environmentNames = items.map((i) => i.environmentName).join(", ");

  const {
    register,
    handleSubmit,
    reset,
    formState: { isSubmitting, errors },
  } = useForm<RenameFormValues>({
    mode: "onChange",
    resolver: zodResolver(renameSchema),
    defaultValues: { key: groupKey },
  });

  const close = () => {
    reset({ key: groupKey });
    onOpenChange(false);
  };

  const onSubmit = async (values: RenameFormValues) => {
    if (values.key === groupKey) {
      close();
      return;
    }
    try {
      await renameMutation.mutateAsync({
        envVarIds: items.map((i) => i.id),
        key: values.key,
      });
      await collection.envVars.utils.refetch();
      toast.success(`Renamed variable to "${values.key}" across ${items.length} environments`);
      close();
    } catch (err) {
      toast.error("Failed to rename variable", {
        description: err instanceof Error ? err.message : undefined,
      });
    }
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => (open ? onOpenChange(true) : close())}
      title="Rename variable"
      subTitle={`Renames "${groupKey}" in all environments (${environmentNames}) so they stay in sync.`}
      footer={
        <div className="flex items-center justify-end gap-2 w-full">
          <Button type="button" variant="outline" size="md" onClick={close} className="px-3">
            Cancel
          </Button>
          <Button
            type="submit"
            form="rename-env-var-form"
            variant="primary"
            size="md"
            className="px-3"
            loading={isSubmitting}
            disabled={isSubmitting}
          >
            Rename
          </Button>
        </div>
      }
    >
      <form id="rename-env-var-form" onSubmit={handleSubmit(onSubmit)}>
        <FormInput
          label="Key"
          className="[&_input]:font-mono"
          placeholder="VARIABLE_NAME"
          error={errors.key?.message}
          {...register("key")}
        />
      </form>
    </DialogContainer>
  );
}
