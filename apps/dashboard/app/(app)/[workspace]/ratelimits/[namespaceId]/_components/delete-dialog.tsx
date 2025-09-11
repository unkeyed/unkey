"use client";
import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, Input } from "@unkey/ui";
import type { PropsWithChildren } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  identifier: z
    .string()
    // biome-ignore lint/suspicious/noSelfCompare: <explanation>
    .refine((v) => v === v, "Please confirm the identifier"),
});

type FormValues = z.infer<typeof formSchema>;

type Props = PropsWithChildren<{
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  overrideId: string;
  identifier: string;
}>;

export const DeleteDialog = ({ isModalOpen, onOpenChange, overrideId, identifier }: Props) => {
  const { register, handleSubmit, watch } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      identifier: "",
    },
  });

  const isValid = watch("identifier") === identifier;

  const onSubmit = async () => {
    collection.ratelimitOverrides.delete(overrideId);
    onOpenChange(false);
  };

  return (
    <DialogContainer
      isOpen={isModalOpen}
      onOpenChange={onOpenChange}
      title="Delete Override"
      footer={
        <div className="w-full flex flex-col gap-2 items-center justify-center">
          <Button
            type="submit"
            form="delete-override-form"
            variant="primary"
            color="danger"
            size="xlg"
            disabled={!isValid}
            className="w-full rounded-lg"
          >
            Delete Override
          </Button>
          <div className="text-gray-9 text-xs">
            This action cannot be undone – proceed with caution
          </div>
        </div>
      }
    >
      <p className="text-gray-11 text-[13px]">
        <span className="font-medium">Warning: </span>
        Are you sure you want to delete this override? The identifier associated with this override
        will now use the default limits.
      </p>

      <form id="delete-override-form" onSubmit={handleSubmit(onSubmit)}>
        <div className="space-y-1">
          <p className="text-gray-11 text-[13px]">
            Type <span className="text-gray-12 font-medium">{identifier}</span> to confirm
          </p>

          <Input {...register("identifier")} placeholder={`Enter "${identifier}" to confirm`} />
        </div>
      </form>
    </DialogContainer>
  );
};
