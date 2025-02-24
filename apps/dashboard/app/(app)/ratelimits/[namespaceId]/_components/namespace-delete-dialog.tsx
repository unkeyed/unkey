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
import { Button, Input } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z
    .string()
    // biome-ignore lint/suspicious/noSelfCompare: <explanation>
    .refine((v) => v === v, "Please confirm the namespace name"),
});

type FormValues = z.infer<typeof formSchema>;
type Props = PropsWithChildren<{
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  namespace: {
    id: string;
    workspaceId: string;
    name: string;
  };
}>;

export const DeleteNamespaceDialog = ({ isModalOpen, onOpenChange, namespace }: Props) => {
  const router = useRouter();
  const {
    register,
    handleSubmit,
    watch,
    formState: { isSubmitting },
  } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
    },
  });

  const isValid = watch("name") === namespace.name;

  const deleteNamespace = trpc.ratelimit.namespace.delete.useMutation({
    onSuccess() {
      toast.success("Namespace Deleted", {
        description: "Your namespace and all its overridden identifiers have been deleted.",
      });
      revalidateTag(tags.namespace(namespace.id));
      router.push("/ratelimits");
      onOpenChange(false);
    },
    onError(err) {
      toast.error("Failed to delete namespace", {
        description: err.message,
      });
    },
  });

  const onSubmit = async () => {
    await deleteNamespace.mutateAsync({ namespaceId: namespace.id });
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
            Delete Namespace
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-4 py-4 px-6 bg-accent-2">
            <p className="text-gray-11 text-[13px]">
              <span className="font-medium">Warning: </span>
              Deleting this namespace while it is in use may cause your current requests to fail.
              You will lose access to analytical data.
            </p>

            <div className="space-y-1">
              <p className="text-gray-11 text-[13px]">
                Type <span className="text-gray-12 font-medium">{namespace.name}</span> to confirm
              </p>

              <Input {...register("name")} placeholder={`Enter "${namespace.name}" to confirm`} />
            </div>
          </div>

          <DialogFooter className="p-6 border-t border-gray-4">
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                variant="destructive"
                disabled={!isValid || deleteNamespace.isLoading || isSubmitting}
                loading={deleteNamespace.isLoading || isSubmitting}
                className="h-10 w-full rounded-lg"
              >
                Delete Namespace
              </Button>
              <div className="text-gray-9 text-xs">
                This action cannot be undone – proceed with caution
              </div>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
