"use client";

import { revalidate } from "@/app/actions";
import { Loading } from "@/components/dashboard/loading";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().trim().min(3, "Name must be at least 3 characters long").max(50),
});

type Props = {
  defaultOpen?: boolean;
};

export const CreateApiButton = ({
  defaultOpen,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const [open, setOpen] = useState(defaultOpen ?? false);
  const router = useRouter();

  const {
    register,
    handleSubmit,
    formState: { errors, isValid, isSubmitting },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const create = trpc.api.create.useMutation({
    async onSuccess(res) {
      toast.success("Your API has been created");
      await revalidate("/apis");
      router.push(`/apis/${res.id}`);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }

  return (
    <>
      <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
        <DialogTrigger asChild>
          <Button variant="primary" {...rest}>
            <Plus />
            Create New API
          </Button>
        </DialogTrigger>
        <DialogContent
          className="bg-gray-1 dark:bg-black drop-shadow-2xl border-gray-4 rounded-lg p-0 gap-0"
          onOpenAutoFocus={(e) => {
            // Prevent auto-focus behavior
            e.preventDefault();
          }}
        >
          <DialogHeader className="border-b border-gray-4">
            <DialogTitle className="px-6 py-4 text-gray-12 font-medium text-base">
              Create New API
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)}>
            <div className="flex flex-col gap-4 p-5 pt-4 bg-accent-2">
              <FormInput
                label="Name"
                description="This is just a human readable name for you and not visible to anyone else"
                error={errors.name?.message}
                {...register("name")}
                placeholder="my-api"
              />
            </div>

            <DialogFooter className="p-6 border-t border-gray-4">
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="submit"
                  variant="primary"
                  disabled={create.isLoading || isSubmitting || !isValid}
                  loading={create.isLoading || isSubmitting}
                  className="h-10 w-full rounded-lg"
                >
                  {create.isLoading ? <Loading /> : "Create API"}
                </Button>
                <div className="text-gray-9 text-xs">
                  You'll be redirected to your new API dashboard after creation
                </div>
              </div>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
};
