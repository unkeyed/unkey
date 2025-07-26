"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Label } from "@/components/ui/label";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import { UnkeyPermission, unkeyPermissionValidation } from "@unkey/rbac";
import { Button, FormInput, toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import type React from "react";
import { useMemo, useState, useCallback } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { PermissionSheet } from "./components/permission-sheet";
import { PermissionBadgeList } from "./components/permission-badge-list";
import { ScrollArea } from "@/components/ui/scroll-area";

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
    defaultOpen = false,
    ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
    const trpcUtils = trpc.useUtils();
    const [isOpen, setIsOpen] = useState(defaultOpen);
    const [selectedPermissions, setSelectedPermissions] = useState<UnkeyPermission[]>([]);
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

    const handlePermissionChange = useCallback((permissions: string[]) => {
        const parsedPermissions = permissions.map((permission) =>
            unkeyPermissionValidation.parse(permission),
        );
        setSelectedPermissions(parsedPermissions);
    }, []);

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
                title="New root key"
                color="default"
                onClick={() => setIsOpen(true)}
            >
                <Plus />
                New root key
            </Button>
            <DynamicDialogContainer
                isOpen={isOpen}
                onOpenChange={setIsOpen}
                title="Create new root key"
                contentClassName="p-0 mb-0 gap-0"
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
                    <div className="flex flex-col p-6 gap-4">
                        <div className="flex flex-col">
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
                            <PermissionSheet selectedPermissions={selectedPermissions} apis={allApis} onChange={handlePermissionChange}>
                                <Button type="button" variant="outline" size="md" className="w-fit rounded-lg pl-3">
                                    Select Permissions...
                                </Button>
                            </PermissionSheet>
                        </div>
                        
                    </div>
                </form>
                <ScrollArea className="w-full overflow-y-auto pt-0 mb-4">
                    <div className="flex flex-col px-6 py-0 gap-3">
                        <PermissionBadgeList
                            selectedPermissions={selectedPermissions}
                            apiId={"workspace"} title="Selected from" name="Workspace"
                            expandCount={3} removePermission={(permission) => handlePermissionChange(selectedPermissions.filter((p) => p !== permission))} />
                        {allApis.map((api) => (
                            <PermissionBadgeList
                                key={api.id}
                                selectedPermissions={selectedPermissions}
                                apiId={api.id} 
                                title="from"
                                name={api.name}
                                expandCount={3} removePermission={(permission) => handlePermissionChange(selectedPermissions.filter((p) => p !== permission))} />
                        ))}
                    </div>
                </ScrollArea>
            </DynamicDialogContainer>
        </>
    );
};
