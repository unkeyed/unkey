"use client";
import { revalidate } from "@/app/actions";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { useTRPC } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus } from "@unkey/icons";
import { Button, FormInput, toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

const formSchema = z.object({
  name: z.string().trim().min(3, "Name must be at least 3 characters long").max(50),
});

type Props = {
  defaultOpen?: boolean;
  workspaceSlug: string;
};

export const CreateApiButton = ({
  defaultOpen,
  workspaceSlug,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const router = useRouter();

  const {
    register,
    handleSubmit,
    formState: { errors, isValid, isSubmitting },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
  });

  const create = useMutation(
    trpc.api.create.mutationOptions({
      async onSuccess(res) {
        toast.success("Your API has been created");
        await revalidate(`/${workspaceSlug}/apis`);
        queryClient.invalidateQueries({
          queryKey: trpc.api.overview.query.queryKey(),
        });
        router.push(`/${workspaceSlug}/apis/${res.id}`);
        setIsOpen(false);
      },
      onError(err) {
        console.error(err);
        toast.error(err.message);
      },
    }),
  );

  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }

  return (
    <>
      <NavbarActionButton
        title="Create new API"
        {...rest}
        color="default"
        onClick={() => setIsOpen(true)}
      >
        <Plus />
        Create new API
      </NavbarActionButton>
      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create New API"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="create-api-form"
              variant="primary"
              size="xlg"
              disabled={create.isPending || isSubmitting || !isValid}
              loading={create.isPending || isSubmitting}
              className="w-full rounded-lg"
            >
              Create API
            </Button>
            <div className="text-gray-9 text-xs">
              You'll be redirected to your new API dashboard after creation
            </div>
          </div>
        }
      >
        <form id="create-api-form" onSubmit={handleSubmit(onSubmit)}>
          <FormInput
            label="Name"
            description="This is just a human readable name for you and not visible to anyone else"
            error={errors.name?.message}
            {...register("name")}
            placeholder="my-api"
          />
        </form>
      </DynamicDialogContainer>
    </>
  );
};
