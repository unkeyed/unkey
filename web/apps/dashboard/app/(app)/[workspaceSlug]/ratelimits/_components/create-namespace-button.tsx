"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import { queryClient, trpcClient } from "@/lib/collections/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import { Button, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useState, useTransition } from "react";
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
  const [isPending, startTransition] = useTransition();

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isValid },
    reset,
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
  });

  const onSubmit = (values: FormValues) => {
    startTransition(async () => {
      try {
        const mutation = trpcClient.ratelimit.namespace.create.mutate({
          name: values.name,
        });

        toast.promise(mutation, {
          loading: "Creating namespace...",
          success: "Namespace created",
          error: (err) => ({
            message: "Failed to create namespace",
            description: err.message,
          }),
        });

        const result = await mutation;

        // Ensure queries are invalidated and refetched before closing
        await queryClient.invalidateQueries({ queryKey: ["ratelimitNamespaces"] });
        await queryClient.refetchQueries({ queryKey: ["ratelimitNamespaces"] });

        reset();
        setIsOpen(false);
      } catch (error) {
        if (error instanceof Error && error.message.includes("already exists")) {
          setError("name", {
            type: "custom",
            message: "Namespace already exists",
          });
        }
      }
    });
  };

  return (
    <>
      <NavbarActionButton
        title="Create new namespace"
        {...rest}
        color="default"
        onClick={() => setIsOpen(true)}
      >
        <Plus iconSize="md-medium" />
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
              disabled={!isValid || isPending}
              className="w-full rounded-lg"
            >
              {isPending ? "Creating..." : "Create Namespace"}
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
