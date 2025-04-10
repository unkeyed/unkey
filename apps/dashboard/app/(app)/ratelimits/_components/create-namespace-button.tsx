"use client";

import { revalidate } from "@/app/actions";
import { DialogContainer } from "@/components/dialog-container";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput } from "@unkey/ui";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import type React from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name must not be empty")
    .max(50, "Name must not exceed 50 characters")
    .regex(
      /^[a-zA-Z0-9_\-\.]+$/,
      "Only alphanumeric characters, underscores, hyphens, and periods are allowed",
    ),
});

type FormValues = z.infer<typeof formSchema>;

export const CreateNamespaceButton = ({
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement>) => {
  const [isOpen, setIsOpen] = useState(false);

  const { ratelimit } = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { errors, isValid, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
  });

  const router = useRouter();

  const create = trpc.ratelimit.namespace.create.useMutation({
    async onSuccess(res) {
      toast.success("Your Namespace has been created");
      router.refresh();
      await revalidate("/ratelimits");
      ratelimit.namespace.query.invalidate();
      router.push(`/ratelimits/${res.id}`);
      setIsOpen(false);
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  const onSubmit = (values: FormValues) => {
    create.mutate(values);
  };

  return (
    <>
      <NavbarActionButton {...rest} color="default" onClick={() => setIsOpen(true)}>
        <Plus size={18} className="w-4 h-4" />
        Create new namespace
      </NavbarActionButton>

      <DialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create Namespace"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="create-namespace-form"
              variant="primary"
              size="xlg"
              disabled={create.isLoading || !isValid || isSubmitting}
              loading={create.isLoading || isSubmitting}
              className="w-full rounded-lg"
            >
              Create Namespace
            </Button>
            <div className="text-gray-9 text-xs">
              Namespaces can be used to separate different rate limiting concerns
            </div>
          </div>
        }
      >
        <form id="create-namespace-form" onSubmit={handleSubmit(onSubmit)}>
          <FormInput
            label="Name"
            description="Alphanumeric, underscores, hyphens or periods are allowed."
            error={errors.name?.message}
            {...register("name")}
            placeholder="email.outbound"
            data-1p-ignore
          />
        </form>
      </DialogContainer>
    </>
  );
};
