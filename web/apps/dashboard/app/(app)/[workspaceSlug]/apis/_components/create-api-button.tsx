"use client";

import { revalidate } from "@/app/actions";
import {
  NavbarActionButton,
  type NavbarActionButtonProps,
} from "@/components/navigation/action-button";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
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
  size?: NavbarActionButtonProps["size"];
};

export const CreateApiButton = ({
  defaultOpen,
  workspaceSlug,
  ...rest
}: Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "size"> & Props) => {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const router = useRouter();
  const { api } = trpc.useUtils();
  const {
    register,
    handleSubmit,
    formState: { errors, isValid, isSubmitting },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
  });

  const create = trpc.api.create.useMutation({
    async onSuccess(res) {
      toast.success("Your keyspace has been created");
      await revalidate(routes.apis.list({ workspaceSlug }));
      api.overview.query.invalidate();
      router.push(routes.apis.detail({ workspaceSlug, apiId: res.id }));
      setIsOpen(false);
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
      <NavbarActionButton
        title="Create new keyspace"
        {...rest}
        color="default"
        onClick={() => setIsOpen(true)}
      >
        <Plus />
        Create new keyspace
      </NavbarActionButton>

      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create New Keyspace"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="create-api-form"
              variant="primary"
              size="xlg"
              disabled={create.isLoading || isSubmitting || !isValid}
              loading={create.isLoading || isSubmitting}
              className="w-full rounded-lg"
            >
              Create Keyspace
            </Button>
            <div className="text-gray-9 text-xs">
              You'll be redirected to your new keyspace dashboard after creation
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
            placeholder="my-keyspace"
          />
        </form>
      </DynamicDialogContainer>
    </>
  );
};
