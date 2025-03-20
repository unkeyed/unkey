"use client";
import { revalidateTag } from "@/app/actions";
import { DialogContainer } from "@/components/dialog-container";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, FormTextarea } from "@unkey/ui";
import { validation } from "@unkey/validation";
import { useRouter } from "next/navigation";
import type { PropsWithChildren, ReactNode } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: validation.name,
  description: validation.description.optional(),
});

type FormValues = z.infer<typeof formSchema>;

type BaseProps = {
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  title: string;
  description?: ReactNode;
  buttonText: string;
  footerText?: string;
  nameDescription?: ReactNode;
  descriptionHelp?: string;
  namePlaceholder?: string;
  descriptionPlaceholder?: string;
  formId: string;
};

type CreateProps = BaseProps & {
  type: "create";
  itemType: "permission" | "role";
  additionalParams?: Record<string, any>;
};

type UpdateProps = BaseProps & {
  type: "update";
  itemType: "permission" | "role";
  item: {
    id: string;
    name: string;
    description?: string | null;
  };
};

type Props = PropsWithChildren<CreateProps | UpdateProps>;

export const RBACForm = (props: Props) => {
  const {
    isModalOpen,
    onOpenChange,
    title,
    buttonText,
    footerText,
    nameDescription,
    descriptionHelp,
    namePlaceholder,
    descriptionPlaceholder,
    formId,
    type,
    itemType,
    children,
  } = props;

  const router = useRouter();
  const { rbac } = trpc.useUtils();

  // Set default values based on form type
  const defaultValues = {
    name: props.type === "update" ? props.item.name : "",
    description: props.type === "update" ? props.item.description ?? "" : "",
  };

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues,
  });

  // Get the appropriate mutation based on form type and item type
  const createPermission = trpc.rbac.createPermission.useMutation({
    onSuccess() {
      toast.success("Permission created", {
        description: "Your new permission has been successfully created",
      });
      handleSuccess();
    },
    onError: handleError,
  });

  const createRole = trpc.rbac.createRole.useMutation({
    onSuccess({ roleId }) {
      toast.success("Role created", {
        description: "Your new role has been successfully created",
      });
      handleSuccess();
      // Special case: navigate to the new role page
      router.push(`/authorization/roles/${roleId}`);
    },
    onError: handleError,
  });

  const updatePermission = trpc.rbac.updatePermission.useMutation({
    onSuccess() {
      toast.success("Permission updated", {
        description: "Permission has been successfully updated",
      });
      if (props.type === "update") {
        revalidateTag(tags.permission(props.item.id));
      }
      handleSuccess();
    },
    onError: handleError,
  });

  const updateRole = trpc.rbac.updateRole.useMutation({
    onSuccess() {
      toast.success("Role updated", {
        description: "Role has been successfully updated",
      });
      if (props.type === "update") {
        revalidateTag(tags.role(props.item.id));
      }
      handleSuccess();
    },
    onError: handleError,
  });

  // Helper functions for success and error handling
  function handleSuccess() {
    reset();
    onOpenChange(false);
    rbac.invalidate();
    router.refresh();
  }

  function handleError(err: any) {
    toast.error(`Failed to ${type} ${itemType}`, {
      description: err.message,
    });
  }

  // Determine loading state based on active mutation
  const isLoading =
    (itemType === "permission" && type === "create" && createPermission.isLoading) ||
    (itemType === "role" && type === "create" && createRole.isLoading) ||
    (itemType === "permission" && type === "update" && updatePermission.isLoading) ||
    (itemType === "role" && type === "update" && updateRole.isLoading) ||
    isSubmitting;

  // Form submission handler
  const onSubmitForm = async (values: FormValues) => {
    try {
      // Carefully handle the description field according to each mutation's requirements
      if (type === "create" && itemType === "permission") {
        await createPermission.mutateAsync({
          name: values.name,
          description: values.description || undefined,
        });
      } else if (type === "create" && itemType === "role") {
        await createRole.mutateAsync({
          name: values.name,
          description: values.description || undefined,
          permissionIds: (props as CreateProps).additionalParams?.permissionIds || [],
        });
      } else if (type === "update" && itemType === "permission") {
        await updatePermission.mutateAsync({
          id: (props as UpdateProps).item.id,
          name: values.name,
          description: values.description || null,
        });
      } else if (type === "update" && itemType === "role") {
        await updateRole.mutateAsync({
          id: (props as UpdateProps).item.id,
          name: values.name,
          description: values.description || null,
        });
      }
    } catch (error) {
      console.error("Form submission error:", error);
    }
  };

  // Default descriptions based on item type
  const defaultNameDescription =
    itemType === "permission" ? (
      <div>
        A unique key to identify your permission. We suggest using{" "}
        <code className="font-bold text-gray-11">.</code> (dot) separated names, to structure your
        hierarchy. For example we use <code className="font-bold text-gray-11">api.create_key</code>{" "}
        or <code className="font-bold text-gray-11">api.update_api</code> in our own permissions.
      </div>
    ) : (
      <div>
        A unique name for your role. You will use this when managing roles through the API. These
        are not customer facing.
      </div>
    );

  const defaultDescriptionHelp =
    itemType === "permission"
      ? "Add a description to help others understand what this permission allows."
      : "Add a description to help others understand what this role represents.";

  const defaultNamePlaceholder = itemType === "permission" ? "domain.create" : "domain.manager";
  const defaultDescriptionPlaceholder =
    itemType === "permission"
      ? "Create a new domain in this account."
      : "Manage domains and DNS records";

  return (
    <>
      {children}
      <DialogContainer
        isOpen={isModalOpen}
        onOpenChange={onOpenChange}
        title={title}
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form={formId}
              variant="primary"
              size="xlg"
              disabled={isLoading}
              loading={isLoading}
              className="w-full rounded-lg"
            >
              {buttonText}
            </Button>
            {footerText && <div className="text-gray-9 text-xs">{footerText}</div>}
          </div>
        }
      >
        <form id={formId} onSubmit={handleSubmit(onSubmitForm)} className="flex flex-col gap-6">
          <FormInput
            label="Name"
            description={nameDescription || defaultNameDescription}
            error={errors.name?.message}
            {...register("name")}
            placeholder={namePlaceholder || defaultNamePlaceholder}
            data-1p-ignore
          />
          <FormTextarea
            label="Description"
            optional
            description={descriptionHelp || defaultDescriptionHelp}
            error={errors.description?.message}
            {...register("description")}
            rows={3}
            placeholder={descriptionPlaceholder || defaultDescriptionPlaceholder}
          />
        </form>
      </DialogContainer>
    </>
  );
};
