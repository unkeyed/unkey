"use client";
import { Label } from "@/components/ui/label";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { CircleInfo } from "@unkey/icons";
import { Button, FormInput, InfoTooltip } from "@unkey/ui";
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
  permissions: z.array(z.string()),
});

type Props = {
  defaultOpen?: boolean;
};

export const CreateRootKeyButton = ({
  defaultOpen,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const router = useRouter();
  const { api } = trpc.useUtils();

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isValid, isSubmitting },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
  });

  const selectedPermissions = watch("permissions");

  // const create = trpc.rootKey.create.useMutation({
  //     onSuccess: async (res) => {
  //         toast.success("Your Root Key has been created");
  //         await revalidate("/settings/root-keys");
  //         api.overview.query.invalidate();
  //         router.push(`/settings/root-keys/${res.keyId}`);
  //         setIsOpen(false);
  //     },
  //     onError: (error) => {
  //         console.error(error);
  //         toast.error(error.message);
  //     },
  // });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    // create.mutate({
    //     name: values.name,
    //     permissions: [],
    // });
  }

  // function handlePermissionChange(permission: string, checked: boolean) {
  //     const current = new Set(selectedPermissions);
  //     if (checked) {
  //         current.add(permission);
  //     } else {
  //         current.delete(permission);
  //     }
  //     setValue("permissions", Array.from(current), { shouldValidate: true });
  // }

  // function handleSelectAll(group: string, checked: boolean) {
  //     const groupPerms = PERMISSIONS.find((g) => g.group === group)?.permissions || [];
  //     const current = new Set(selectedPermissions);
  //     if (checked) {
  //         groupPerms.forEach((p) => current.add(p));
  //     } else {
  //         groupPerms.forEach((p) => current.delete(p));
  //     }
  //     setValue("permissions", Array.from(current), { shouldValidate: true });
  // }

  // function isAllSelected(group: string) {
  //     const groupPerms = PERMISSIONS.find((g) => g.group === group)?.permissions || [];
  //     return groupPerms.every((p) => selectedPermissions.includes(p));
  // }

  return (
    <>
      <Button
        variant="primary"
        size="md"
        className="px-3 rounded-lg"
        onClick={() => setIsOpen(true)}
      >
        New root key
      </Button>
      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create new root key"
        className="max-w-[460px]"
        subTitle="Define a new root key and assign permissions"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="create-rootkey-form"
              variant="primary"
              size="xlg"
              disabled={true}
              // disabled={create.isLoading || isSubmitting || !isValid}
              // loading={create.isLoading || isSubmitting}
              className="w-full rounded-lg"
            >
              Create root key
            </Button>
            <div className="text-gray-9 text-xs">This root key will be created immediately</div>
          </div>
        }
      >
        <form id="create-rootkey-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label className="text-[13px] font-regular text-gray-11">
                Root Key
                <InfoTooltip content={"Internal name for your reference"}>
                  <CircleInfo className="inline-block ml-2 text-gray-8" size="md-regular" />
                </InfoTooltip>
              </Label>
              <FormInput
                error={errors.name?.message}
                {...register("name")}
                placeholder="key-name"
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label className="text-[13px] font-regular text-gray-11">
                Permissions
                <InfoTooltip
                  content={
                    "Permissions are the actions that the root key can perform. You can select multiple permissions to grant the root key access to multiple actions."
                  }
                >
                  <CircleInfo className="inline-block ml-2 text-gray-8" size="md-regular" />
                </InfoTooltip>
              </Label>
              <Button type="button" variant="outline" size="lg" className="w-fit rounded-lg pl-3">
                Select Permissions...
              </Button>
            </div>
          </div>
        </form>
        {/* Drawer for permissions selection */}
      </DynamicDialogContainer>
    </>
  );
};

// If @unkey/ui does not provide a Drawer, you can use Radix UI's Drawer or implement a simple one. If needed, let me know to provide a custom Drawer implementation.
