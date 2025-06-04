"use client";
import { DialogContainer } from "@/components/dialog-container";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { PenWriting3, Plus } from "@unkey/icons";
import { Button, FormInput, FormTextarea } from "@unkey/ui";
import { useEffect, useState } from "react";
import { Controller, FormProvider } from "react-hook-form";
import { KeyField } from "./components/assign-key/key-field";
import { PermissionField } from "./components/assign-permission/permissions-field";
import { useUpsertRole } from "./hooks/use-upsert-role";
import { type FormValues, rbacRoleSchema } from "./upsert-role.schema";

const FORM_STORAGE_KEY = "unkey_upsert_role_form_state";

const getDefaultValues = (): Partial<FormValues> => ({
  roleName: "",
  roleDescription: "",
  keyIds: [],
  permissionIds: [],
});

type UpsertRoleDialogProps = {
  roleId?: string;
  existingRole?: {
    name: string;
    description?: string;
    keyIds: string[];
    permissionIds?: string[];
  };
  triggerButton?: React.ReactNode;
  isOpen?: boolean;
  onClose?: () => void;
  selectedKeysData?: { keyId: string; keyName: string | null }[];
  selectedPermissionsData?: {
    id: string;
    name: string;
    slug: string;
    description: string | null;
  }[];
};

export const UpsertRoleDialog = ({
  roleId,
  existingRole,
  triggerButton,
  isOpen: externalIsOpen,
  onClose: externalOnClose,
  selectedKeysData,
  selectedPermissionsData,
}: UpsertRoleDialogProps) => {
  const [internalIsOpen, setInternalIsOpen] = useState(false);
  const isEditMode = Boolean(roleId);

  // Use external state if provided, otherwise use internal state
  const isDialogOpen = externalIsOpen !== undefined ? externalIsOpen : internalIsOpen;
  const setIsDialogOpen =
    externalOnClose !== undefined
      ? (open: boolean) => !open && externalOnClose()
      : setInternalIsOpen;

  // Use different storage keys for create vs edit to avoid conflicts
  const storageKey = isEditMode ? `${FORM_STORAGE_KEY}_edit_${roleId}` : FORM_STORAGE_KEY;

  const methods = usePersistedForm<FormValues>(
    storageKey,
    {
      resolver: zodResolver(rbacRoleSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getDefaultValues(),
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
    control,
  } = methods;

  const upsertRoleMutation = useUpsertRole(() => {
    clearPersistedData();
    reset(getDefaultValues());
    setIsDialogOpen(false);
  });

  // Load existing role data when in edit mode
  useEffect(() => {
    if (isEditMode && existingRole) {
      const editValues: Partial<FormValues> = {
        roleName: existingRole.name,
        roleDescription: existingRole.description || "",
        keyIds: existingRole.keyIds || null,
        permissionIds: existingRole.permissionIds || [],
      };
      reset(editValues);
    }
  }, [isEditMode, existingRole, reset]);

  // Add invisible roleId field for updates
  const hiddenRoleIdRegister = isEditMode ? register("roleId") : null;

  const onSubmit = async (data: FormValues) => {
    const mutationData = {
      ...data,
      ...(isEditMode && { roleId }), // Include roleId only for updates
    };

    upsertRoleMutation.mutate(mutationData);
  };

  const handleDialogClose = (open: boolean) => {
    if (!open) {
      saveCurrentValues();
      if (!isEditMode) {
        reset(getDefaultValues());
      }
    }
    setIsDialogOpen(open);
  };

  const dialogConfig = {
    title: isEditMode ? "Edit role" : "Create new role",
    subtitle: isEditMode
      ? "Update role settings and permissions"
      : "Define a role and assign permissions",
    buttonText: isEditMode ? "Update role" : "Create new role",
    footerText: isEditMode
      ? "Changes will be applied immediately"
      : "This role will be created immediately",
    triggerTitle: isEditMode ? "Edit role" : "Create new role",
  };

  const defaultTrigger = (
    <NavbarActionButton title={dialogConfig.triggerTitle} onClick={() => setIsDialogOpen(true)}>
      {isEditMode ? <PenWriting3 /> : <Plus />}
      {dialogConfig.triggerTitle}
    </NavbarActionButton>
  );

  return (
    <>
      {!triggerButton && <Navbar.Actions>{defaultTrigger}</Navbar.Actions>}

      {triggerButton && (
        // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
        <div onClick={() => setIsDialogOpen(true)}>{triggerButton}</div>
      )}

      <FormProvider {...methods}>
        <form id="upsert-role-form" onSubmit={handleSubmit(onSubmit)}>
          {/* Hidden field for role ID in edit mode */}
          {isEditMode && hiddenRoleIdRegister && (
            <input type="hidden" value={roleId} {...hiddenRoleIdRegister} />
          )}

          <DialogContainer
            title={dialogConfig.title}
            subTitle={dialogConfig.subtitle}
            isOpen={isDialogOpen}
            onOpenChange={handleDialogClose}
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="submit"
                  form="upsert-role-form"
                  variant="primary"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!isValid || upsertRoleMutation.isLoading}
                  loading={upsertRoleMutation.isLoading}
                >
                  {dialogConfig.buttonText}
                </Button>
                <div className="text-gray-9 text-xs">{dialogConfig.footerText}</div>
              </div>
            }
          >
            <div className="space-y-5 px-2 py-1">
              {/* Role Name - Required */}
              <FormInput
                className="[&_input:first-of-type]:h-[36px]"
                placeholder="domain.manager"
                label="Name"
                maxLength={60}
                description="A unique name for your role. You will use this when managing roles through the API. These are not customer facing."
                error={errors.roleName?.message}
                variant="default"
                required
                {...register("roleName")}
              />

              {/* Role Description - Optional */}
              <FormTextarea
                className="[&_input:first-of-type]:h-[36px]"
                label="Description"
                placeholder="Manage domains and DNS records"
                maxLength={30}
                description="Add a description to help others understand what this role represents."
                error={errors.roleDescription?.message}
                optional
                {...register("roleDescription")}
              />

              {/* Key Selection */}
              <Controller
                name="keyIds"
                control={control}
                render={({ field, fieldState }) => (
                  <KeyField
                    value={field.value || []}
                    onChange={field.onChange}
                    error={fieldState.error?.message}
                    selectedKeysData={selectedKeysData}
                  />
                )}
              />

              {/* Permission Selection */}
              <Controller
                name="permissionIds"
                control={control}
                render={({ field, fieldState }) => (
                  <PermissionField
                    value={field.value || []}
                    onChange={field.onChange}
                    error={fieldState.error?.message}
                    selectedPermissionsData={selectedPermissionsData}
                  />
                )}
              />
            </div>
          </DialogContainer>
        </form>
      </FormProvider>
    </>
  );
};
