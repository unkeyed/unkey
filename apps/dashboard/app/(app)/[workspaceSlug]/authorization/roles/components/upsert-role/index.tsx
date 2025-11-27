"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import type {
  RoleKey,
  RolePermission,
} from "@/lib/trpc/routers/authorization/roles/connected-keys-and-perms";
import { zodResolver } from "@hookform/resolvers/zod";
import { PenWriting3, Plus } from "@unkey/icons";
import { Button, DialogContainer, FormInput, FormTextarea } from "@unkey/ui";
import { useEffect, useState } from "react";
import { Controller, FormProvider } from "react-hook-form";
import { useRoleLimits } from "../table/hooks/use-role-limits";
import { KeyField } from "./components/assign-key/key-field";
import { PermissionField } from "./components/assign-permission/permissions-field";
import { useUpsertRole } from "./hooks/use-upsert-role";
import { type FormValues, rbacRoleSchema } from "./upsert-role.schema";

const FORM_STORAGE_KEY = "unkey_upsert_role_form_state";

type ExistingRole = {
  id: string;
  name: string;
  description?: string;
  keyIds: string[];
  permissionIds?: string[];
  assignedKeysDetails: RoleKey[];
  assignedPermsDetails: RolePermission[];
};

const getDefaultValues = (existingRole?: ExistingRole): Partial<FormValues> => {
  if (existingRole) {
    return {
      roleId: existingRole.id,
      roleName: existingRole.name,
      roleDescription: existingRole.description || "",
      keyIds: existingRole.keyIds || [],
      permissionIds: existingRole.permissionIds || [],
    };
  }

  return {
    roleName: "",
    roleDescription: "",
    keyIds: [],
    permissionIds: [],
  };
};

type UpsertRoleDialogProps = {
  existingRole?: ExistingRole;
  triggerButton?: boolean;
  isOpen?: boolean;
  onClose?: () => void;
};

export const UpsertRoleDialog = ({
  existingRole,
  triggerButton,
  isOpen: externalIsOpen,
  onClose: externalOnClose,
}: UpsertRoleDialogProps) => {
  const [internalIsOpen, setInternalIsOpen] = useState(false);
  const isEditMode = Boolean(existingRole?.id);

  const { calculateLimits } = useRoleLimits(existingRole?.id);

  // Use external state if provided, otherwise use internal state
  const isDialogOpen = externalIsOpen !== undefined ? externalIsOpen : internalIsOpen;
  const setIsDialogOpen =
    externalOnClose !== undefined
      ? (open: boolean) => !open && externalOnClose()
      : setInternalIsOpen;

  const storageKey = isEditMode ? `${FORM_STORAGE_KEY}_edit_${existingRole?.id}` : FORM_STORAGE_KEY;

  const methods = usePersistedForm<FormValues>(
    storageKey,
    {
      resolver: zodResolver(rbacRoleSchema),
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
    control,
  } = methods;

  useEffect(() => {
    if (!isDialogOpen) {
      return;
    }

    const loadData = async () => {
      if (existingRole) {
        // Edit mode
        const hasSavedData = await loadSavedValues();
        if (!hasSavedData) {
          const editValues = getDefaultValues(existingRole);
          reset(editValues);
        }
      } else {
        // Create mode
        const hasSavedData = await loadSavedValues();
        if (!hasSavedData) {
          reset(getDefaultValues());
        }
      }
    };

    loadData();
  }, [existingRole, reset, loadSavedValues, isDialogOpen]);

  const upsertRoleMutation = useUpsertRole(() => {
    clearPersistedData();
    reset(getDefaultValues());
    setIsDialogOpen(false);
  });

  const onSubmit = async (data: FormValues) => {
    // Calculate limits with current form data
    const { hasKeyWarning, hasPermWarning } = calculateLimits(data.keyIds, data.permissionIds);

    const submissionData: FormValues = {
      ...data,
      keyIds: hasKeyWarning ? undefined : data.keyIds,
      permissionIds: hasPermWarning ? undefined : data.permissionIds,
    };

    upsertRoleMutation.mutate(submissionData);
  };

  const handleDialogToggle = (open: boolean) => {
    if (!open) {
      saveCurrentValues();
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
      {triggerButton && <Navbar.Actions>{defaultTrigger}</Navbar.Actions>}
      <FormProvider {...methods}>
        <form id={`upsert-role-form-${existingRole?.id}`} onSubmit={handleSubmit(onSubmit)}>
          {/* Hidden input for roleId in edit mode */}
          {isEditMode && <input type="hidden" {...register("roleId")} />}
          <DialogContainer
            title={dialogConfig.title}
            subTitle={dialogConfig.subtitle}
            isOpen={isDialogOpen}
            onOpenChange={handleDialogToggle}
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="submit"
                  form={`upsert-role-form-${existingRole?.id}`}
                  variant="primary"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!isValid || upsertRoleMutation.isPending}
                  loading={upsertRoleMutation.isPending}
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
                placeholder="domain.manager"
                label="Name"
                maxLength={60}
                description="A unique name for your role. You will use this when managing roles through the API. These are not customer facing."
                error={errors.roleName?.message}
                variant="default"
                required
                {...register("roleName")}
              />

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

              <Controller
                name="keyIds"
                control={control}
                render={({ field, fieldState }) => (
                  <KeyField
                    roleId={existingRole?.id}
                    value={field.value ?? []}
                    onChange={field.onChange}
                    error={fieldState.error?.message}
                    assignedKeyDetails={existingRole?.assignedKeysDetails ?? []}
                  />
                )}
              />

              <Controller
                name="permissionIds"
                control={control}
                render={({ field, fieldState }) => (
                  <PermissionField
                    roleId={existingRole?.id}
                    value={field.value ?? []}
                    onChange={field.onChange}
                    error={fieldState.error?.message}
                    assignedPermsDetails={existingRole?.assignedPermsDetails ?? []}
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
