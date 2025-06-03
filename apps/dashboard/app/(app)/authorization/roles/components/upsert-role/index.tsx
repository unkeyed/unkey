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
import { type FormValues, rbacRoleSchema } from "./upsert-role.schema";

const FORM_STORAGE_KEY = "unkey_upsert_role_form_state";

const getDefaultValues = (): Partial<FormValues> => ({
  roleName: "",
  roleDescription: "",
  roleSlug: "",
  keyIds: [],
  permissionIds: [],
});

interface UpsertRoleDialogProps {
  roleId?: string;
  existingRole?: {
    name: string;
    slug: string;
    description?: string;
    keyIds: string[];
    permissionIds?: string[];
  };
  triggerButton?: React.ReactNode;
}

export const UpsertRoleDialog = ({
  roleId,
  existingRole,
  triggerButton,
}: UpsertRoleDialogProps) => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const isEditMode = Boolean(roleId);

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

  // Load existing role data when in edit mode
  useEffect(() => {
    if (isEditMode && existingRole) {
      const editValues: Partial<FormValues> = {
        roleName: existingRole.name,
        roleSlug: existingRole.slug,
        roleDescription: existingRole.description || "",
        keyIds: existingRole.keyIds || null, // Changed to single key
        permissionIds: existingRole.permissionIds || [],
      };
      reset(editValues);
    }
  }, [isEditMode, existingRole, reset]);

  // Add invisible roleId field for updates
  const hiddenRoleIdRegister = isEditMode ? register("roleId") : null;

  const onSubmit = async (data: FormValues) => {
    try {
      if (isEditMode) {
        console.log("Updating role:", { ...data, roleId });
        // TODO: await updateRole(roleId, data);
      } else {
        console.log("Creating role:", data);
        // TODO: await createRole(data);
      }

      clearPersistedData();
      reset(getDefaultValues());
      setIsDialogOpen(false);
    } catch (error) {
      console.error(`Failed to ${isEditMode ? "update" : "create"} role:`, error);
      // TODO: Show error toast
    }
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
                  disabled={!isValid}
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
                placeholder="Domain manager"
                label="Name"
                maxLength={64}
                description="A descriptive name for this role (2-64 characters, must start with a letter)"
                error={errors.roleName?.message}
                variant="default"
                required
                {...register("roleName")}
              />

              {/* Role Slug - Auto-generated but editable */}
              <FormInput
                className="[&_input:first-of-type]:h-[36px]"
                label="Slug"
                placeholder="domain.manager"
                maxLength={30}
                description="A unique name for your role. You will use this when managing roles through the API. These are not customer facing."
                error={errors.roleSlug?.message}
                required
                {...register("roleSlug")}
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
                    value={field.value || null}
                    onChange={field.onChange}
                    error={fieldState.error?.message}
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
