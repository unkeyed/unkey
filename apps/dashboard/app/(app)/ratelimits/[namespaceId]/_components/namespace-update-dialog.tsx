"use client";

import { revalidateTag } from "@/app/actions";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { CircleInfo } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { validation } from "@unkey/validation";
import { useRouter } from "next/navigation";
import type { PropsWithChildren, ReactNode } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { InputTooltip } from "../_overview/components/table/components/logs-actions/components/input-tooltip";

const formSchema = z.object({
  name: validation.name,
  namespaceId: validation.unkeyId,
});

type FormValues = z.infer<typeof formSchema>;

type FormFieldProps = {
  label: string;
  tooltip?: string;
  error?: string;
  children: ReactNode;
};

const FormField = ({ label, tooltip, error, children }: FormFieldProps) => (
  // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
  <div className="flex flex-col gap-1" onClick={(e) => e.stopPropagation()}>
    <Label
      className="text-gray-11 text-[13px] flex items-center"
      onClick={(e) => e.preventDefault()}
    >
      {label}
      {tooltip && (
        <InputTooltip desc={tooltip}>
          <CircleInfo size="md-regular" className="text-accent-8 ml-[10px]" />
        </InputTooltip>
      )}
    </Label>
    {children}
    {error && <span className="text-error-10 text-[13px] font-medium">{error}</span>}
  </div>
);

type Props = PropsWithChildren<{
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  namespace: {
    id: string;
    name: string;
    workspaceId: string;
  };
}>;

export const NamespaceUpdateNameDialog = ({ isModalOpen, onOpenChange, namespace }: Props) => {
  const router = useRouter();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: namespace.name,
      namespaceId: namespace.id,
    },
  });

  const updateName = trpc.ratelimit.namespace.update.name.useMutation({
    onSuccess() {
      toast.success("Your namespace name has been renamed!");
      revalidateTag(tags.namespace(namespace.id));
      router.refresh();
      onOpenChange(false);
    },
    onError(err) {
      toast.error("Failed to update namespace name", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: FormValues) => {
    if (values.name === namespace.name || !values.name) {
      return toast.error("Please provide a different name before saving.");
    }
    await updateName.mutateAsync({
      name: values.name,
      namespaceId: namespace.id,
    });
  };

  return (
    <Dialog open={isModalOpen} onOpenChange={onOpenChange}>
      <DialogContent
        className="bg-gray-1 dark:bg-black drop-shadow-2xl border-gray-4 rounded-lg p-0 gap-0"
        onOpenAutoFocus={(e) => {
          e.preventDefault();
        }}
      >
        <DialogHeader className="border-b border-gray-4">
          <DialogTitle className="px-6 py-4 text-gray-12 font-medium text-base">
            Update Namespace Name
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-4 py-4 px-6 bg-accent-2">
            <FormField
              label="Name"
              tooltip="Update the name of your namespace"
              error={errors.name?.message}
            >
              <Input
                {...register("name")}
                placeholder="Enter namespace name"
                className="border border-gray-4 focus:border focus:border-gray-4 px-3 py-1 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4 rounded-md"
              />
              <input type="hidden" {...register("namespaceId")} defaultValue={namespace.id} />
            </FormField>
          </div>
          <DialogFooter className="px-6 py-4 border-t border-gray-4">
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                variant="primary"
                disabled={updateName.isLoading || isSubmitting}
                loading={updateName.isLoading || isSubmitting}
                className="h-10 w-full rounded-lg"
              >
                Update Namespace
              </Button>
              <div className="text-gray-9 text-xs">Name changes are applied immediately</div>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
