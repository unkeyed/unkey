"use client";

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
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
  const router = useRouter();
  const { ratelimit } = trpc.useUtils();

  const {
    register,
    handleSubmit,
    watch,
    formState: { isSubmitting },
  } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      identifier: "",
    },
  });

  const isValid = watch("identifier") === identifier;

  const deleteOverride = trpc.ratelimit.override.delete.useMutation({
    onSuccess() {
      toast.success("Override has been deleted", {
        description: "Changes may take up to 60s to propagate globally",
      });
      onOpenChange(false);
      router.push("/ratelimits/");
      ratelimit.overview.logs.query.invalidate();
    },
    onError(err) {
      toast.error("Failed to delete override", {
        description: err.message,
      });
    },
  });

  const onSubmit = async () => {
    try {
      await deleteOverride.mutateAsync({ id: overrideId });
    } catch (error) {
      console.error("Delete error:", error);
    }
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
            Delete Override
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-4 py-4 px-6 bg-accent-2">
            <p className="text-gray-11 text-[13px]">
              <span className="font-medium">Warning: </span>
              Are you sure you want to delete this override? The identifier associated with this
              override will now use the default limits.
            </p>

            <div className="space-y-1">
              <p className="text-gray-11 text-[13px]">
                Type <span className="text-gray-12 font-medium">{identifier}</span> to confirm
              </p>

              <Input
                {...register("identifier")}
                placeholder={`Enter "${identifier}" to confirm`}
                className="border border-gray-5 focus:border focus:border-gray-4 px-3 py-1 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4 rounded-md placeholder:text-gray-8 h-9"
              />
            </div>
          </div>

          <DialogFooter className="p-6 border-t border-gray-4">
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                variant="destructive"
                disabled={!isValid || deleteOverride.isLoading || isSubmitting}
                loading={deleteOverride.isLoading || isSubmitting}
                className="h-10 w-full rounded-lg"
              >
                Delete Override
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
