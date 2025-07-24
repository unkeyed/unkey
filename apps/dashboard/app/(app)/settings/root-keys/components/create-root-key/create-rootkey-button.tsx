"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Label } from "@/components/ui/label";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { Button, FormInput, toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import type React from "react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { PermissionSheet } from "./components/permission-sheet";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

const DEFAULT_LIMIT = 10;

const formSchema = z.object({
  name: z.string().trim().min(3, "Name must be at least 3 characters long").max(50),
  permissions: z.array(unkeyPermissionValidation).min(1, "At least one permission is required"),
});

type Props = {
  defaultOpen?: boolean;
};

export const CreateRootKeyButton = ({
  defaultOpen,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const trpcUtils = trpc.useUtils();
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);

  const {
    data: apisData,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = trpc.api.overview.query.useInfiniteQuery(
    { limit: DEFAULT_LIMIT },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  const allApis = useMemo(() => {
    if (!apisData?.pages) {
      return [];
    }
    return apisData.pages.flatMap((page) => {
      return page.apiList.map((api) => ({
        id: api.id,
        name: api.name,
      }));
    });
  }, [apisData]);

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

  // const selectedPermissions = watch("permissions");

  const key = trpc.rootKey.create.useMutation({
    onSuccess() {
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await key.mutateAsync({
      name: values.name,
      permissions: values.permissions,
    });
    setIsOpen(false);
  }

  const handlePermissionChange = (permissions: string[]) => {
    const parsedPermissions = permissions.map((permission) =>
      unkeyPermissionValidation.parse(permission),
    );
    setValue("permissions", parsedPermissions);
  };

  return (
    <>
      <NavbarActionButton
        title="New root key"
        {...rest}
        color="default"
        onClick={() => setIsOpen(true)}
      >
        <Plus />
        New root key
      </NavbarActionButton>
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
              <FormInput
                label="Name"
                description="Give your key a name, this is not customer facing."
                error={errors.name?.message}
                {...register("name")}
                placeholder="key-name"
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label className="text-[13px] font-regular text-gray-10">Permissions</Label>
              <PermissionSheet apis={allApis} onChange={handlePermissionChange}>
                <Button type="button" variant="outline" size="md" className="w-fit rounded-lg pl-3">
                  Select Permissions...
                </Button>
              </PermissionSheet>
            </div>
          </div>
        </form>
      </DynamicDialogContainer>
    </>
  );
};
