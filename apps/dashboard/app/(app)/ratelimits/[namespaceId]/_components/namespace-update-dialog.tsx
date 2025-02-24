"use client";

import { revalidateTag } from "@/app/actions";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput } from "@unkey/ui";
import { validation } from "@unkey/validation";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";

import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: validation.name,
  namespaceId: validation.unkeyId,
});

type FormValues = z.infer<typeof formSchema>;

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
        className="bg-gray-1 dark:bg-black drop-shadow-2xl border-gray-4 rounded-2xl p-0 gap-0"
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
            <FormInput
              label="Name"
              description="Update the name of your namespace"
              error={errors.name?.message}
              placeholder="Enter namespace name"
              {...register("name")}
            />
            <input type="hidden" {...register("namespaceId")} defaultValue={namespace.id} />
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
