"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { PenWriting3, Plus } from "@unkey/icons";
import { Button, DialogContainer, FormInput, FormTextarea } from "@unkey/ui";
import { useEffect, useState } from "react";
import { FormProvider } from "react-hook-form";
import { useUpsertPermission } from "./hooks/use-upsert-permission";
import { type PermissionFormValues, permissionSchema } from "./upsert-permission.schema";

const FORM_STORAGE_KEY = "unkey_upsert_permission_form_state";

type ExistingPermission = {
  id: string;
  name: string;
  slug: string;
  description?: string;
};

const getDefaultValues = (
  existingPermission?: ExistingPermission,
): Partial<PermissionFormValues> => {
  if (existingPermission) {
    return {
      permissionId: existingPermission.id,
      name: existingPermission.name,
      slug: existingPermission.slug,
      description: existingPermission.description || "",
    };
  }

  return {
    name: "",
    slug: "",
    description: "",
  };
};

type UpsertPermissionDialogProps = {
  existingPermission?: ExistingPermission;
  triggerButton?: boolean;
  isOpen?: boolean;
  onClose?: () => void;
};

export const UpsertPermissionDialog = ({
  existingPermission,
  triggerButton,
  isOpen: externalIsOpen,
  onClose: externalOnClose,
}: UpsertPermissionDialogProps) => {
  const [internalIsOpen, setInternalIsOpen] = useState(false);
  const isEditMode = Boolean(existingPermission?.id);

  // Use external state if provided, otherwise use internal state
  const isDialogOpen = externalIsOpen !== undefined ? externalIsOpen : internalIsOpen;
  const setIsDialogOpen =
    externalOnClose !== undefined
      ? (open: boolean) => !open && externalOnClose()
      : setInternalIsOpen;

  const storageKey = isEditMode
    ? `${FORM_STORAGE_KEY}_edit_${existingPermission?.id}`
    : FORM_STORAGE_KEY;

  const formId = `upsert-permission-form-${existingPermission?.id || "new"}`;

  const methods = usePersistedForm<PermissionFormValues>(
    storageKey,
    {
      resolver: zodResolver(permissionSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
    },
    "memory",
  );

  const {
    register,
    formState: { errors, isValid },
    handleSubmit,
    reset,
    clearPersistedData,
    saveCurrentValues,
    loadSavedValues,
  } = methods;

  useEffect(() => {
    if (!isDialogOpen) {
      return;
    }

    const loadData = async () => {
      const hasSavedData = await loadSavedValues();
      if (!hasSavedData) {
        reset(getDefaultValues(existingPermission));
      }
    };

    loadData();
  }, [existingPermission, reset, loadSavedValues, isDialogOpen]);

  const upsertPermissionMutation = useUpsertPermission(() => {
    clearPersistedData();
    reset(getDefaultValues());
    setIsDialogOpen(false);
  });

  const onSubmit = async (data: PermissionFormValues) => {
    if (isEditMode && !data.permissionId) {
      console.error("Edit mode requires permissionId");
      return;
    }

    upsertPermissionMutation.mutate(data);
  };

  const handleDialogToggle = (open: boolean) => {
    if (!open) {
      saveCurrentValues();
    }
    setIsDialogOpen(open);
  };

  const dialogConfig = {
    title: isEditMode ? "Edit permission" : "Create new permission",
    subtitle: isEditMode
      ? "Update permission details"
      : "Define a new permission for your application",
    buttonText: isEditMode ? "Update permission" : "Create new permission",
    footerText: isEditMode
      ? "Changes will be applied immediately"
      : "This permission will be created immediately",
    triggerTitle: isEditMode ? "Edit permission" : "Create new permission",
  };

  const defaultTrigger = (
    <NavbarActionButton title={dialogConfig.triggerTitle} onClick={() => setIsDialogOpen(true)}>
      {isEditMode ? <PenWriting3 /> : <Plus />}
      {dialogConfig.triggerTitle}
    </NavbarActionButton>
  );

  return (
    <>
      {triggerButton && <Navbar.Actions>{defaultTrigger}</Navbar.Actions>}
      <FormProvider {...methods}>
        <form id={formId} onSubmit={handleSubmit(onSubmit)}>
          {/* Hidden input for permissionId in edit mode */}
          {isEditMode && existingPermission?.id && (
            <input type="hidden" {...register("permissionId")} />
          )}
          <DialogContainer
            title={dialogConfig.title}
            subTitle={dialogConfig.subtitle}
            isOpen={isDialogOpen}
            onOpenChange={handleDialogToggle}
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="submit"
                  form={formId}
                  variant="primary"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!isValid || upsertPermissionMutation.isPending}
                  loading={upsertPermissionMutation.isPending}
                >
                  {dialogConfig.buttonText}
                </Button>
                <div className="text-gray-9 text-xs">{dialogConfig.footerText}</div>
              </div>
            }
          >
            <div className="space-y-5 px-2 py-1">
              <FormInput
                className="[&_input:first-of-type]:h-[36px]"
                placeholder="Manage Domains"
                label="Name"
                maxLength={60}
                description="A human-readable name for this permission that describes what it allows."
                error={errors.name?.message}
                variant="default"
                required
                {...register("name")}
              />

              <FormInput
                className="[&_input:first-of-type]:h-[36px]"
                placeholder="manage.domains"
                label="Slug"
                maxLength={50}
                description="A unique identifier used in code."
                error={errors.slug?.message}
                variant="default"
                required
                {...register("slug")}
              />

              <FormTextarea
                className="[&_input:first-of-type]:h-[36px]"
                label="Description"
                placeholder="Allows user to create, update, and delete domain configurations and DNS records"
                maxLength={200}
                description="Add a detailed description to help others understand what this permission grants access to."
                error={errors.description?.message}
                optional
                {...register("description")}
              />
            </div>
          </DialogContainer>
        </form>
      </FormProvider>
    </>
  );
};
