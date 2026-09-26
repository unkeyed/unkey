"use client";

import { revalidate } from "@/app/actions";
import { useProjectScope } from "@/hooks/use-project-scope";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { IconPlusOutline18 } from "@unkey/icons";
import { Button, FormInput, toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
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

export function CreateApiButton({ defaultOpen, workspaceSlug }: Props) {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const scope = useProjectScope();
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

  const create = useMutation({
    mutationFn: async (values: z.infer<typeof formSchema>) => {
      const response = await getUnkeyClient().apis.createApi(values);
      return response.data;
    },
    async onSuccess(data) {
      toast.success("Your keyspace has been created");
      await revalidate(routes.apis.list({ workspaceSlug, ...scope }));
      api.overview.query.invalidate();
      router.push(routes.apis.detail({ workspaceSlug, ...scope, apiId: data.apiId }));
      setIsOpen(false);
    },
    onError(err) {
      const { message, description } = getErrorToast(err, "Failed to Create Keyspace");
      toast.error(message, { description });
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }

  return (
    <>
      <Button size="md" variant="primary" onClick={() => setIsOpen(true)}>
        <IconPlusOutline18 />
        Create keyspace
      </Button>

      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create keyspace"
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
}
