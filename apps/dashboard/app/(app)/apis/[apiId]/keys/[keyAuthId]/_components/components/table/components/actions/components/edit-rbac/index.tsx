"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { usePersistedForm } from "@/hooks/use-persisted-form";

import { trpc } from "@/lib/trpc/client";
import type { KeyPermission, KeyRole } from "@/lib/trpc/routers/key/rbac/connected-roles-and-perms";
import { zodResolver } from "@hookform/resolvers/zod";
import { HandHoldingKey, PenWriting3 } from "@unkey/icons";
import { Button, DialogContainer } from "@unkey/ui";
import { useEffect, useMemo, useState } from "react";
import { Controller, FormProvider } from "react-hook-form";
import { useUpdateKeyRbac } from "../hooks/use-edit-rbac";
import { KeyInfo } from "../key-info";
import { PermissionField } from "./components/assign-permission/permissions-field";
import { RoleField } from "./components/assign-role/role-field";
import { useFetchPermissionSlugs } from "./components/hooks/use-fetch-permission-slugs";
import { type FormValues, updateKeyRbacSchema } from "./update-key-rbac.schema";

const FORM_STORAGE_KEY = "unkey_key_rbac_form_state";

type ExistingKey = {
  id: string;
  name?: string;
  roleIds: string[];
  permissionIds: string[];
};

const DIALOG_CONFIG = {
  title: "Manage roles and permissions",
  subtitle: "Assign roles or permissions to control what this key can do",
  buttonText: "Update roles and permissions",
  footerText: "This key will be updated immediately",
  triggerTitle: "Manage roles and permissions",
};

type KeyRbacDialogProps = {
  existingKey: ExistingKey;
  triggerButton?: boolean;
  isOpen?: boolean;
  onClose?: () => void;
};

const getDefaultValues = (
  existingKey: ExistingKey,
  apiData?: { roles: KeyRole[]; permissions: KeyPermission[] },
): FormValues => {
  return {
    keyId: existingKey.id,
    roleIds: apiData?.roles.map((r) => r.id) ?? existingKey.roleIds ?? [],
    permissionIds: apiData?.permissions.map((p) => p.id) ?? existingKey.permissionIds ?? [],
  };
};

const GrantedAccess = ({
  slugs,
  totalCount,
  isLoading,
}: {
  slugs?: string[];
  totalCount?: number;
  isLoading: boolean;
}) => {
  const [displaySlugs, setDisplaySlugs] = useState<string[]>([]);
  const [displayCount, setDisplayCount] = useState(0);

  useEffect(() => {
    if (!isLoading && slugs) {
      const timer = setTimeout(() => {
        setDisplaySlugs(slugs);
        setDisplayCount(totalCount || 0);
      }, 150);
      return () => clearTimeout(timer);
    }
  }, [slugs, totalCount, isLoading]);

  const memoizedSlugs = useMemo(() => {
    return displaySlugs.map((slug) => (
      <div
        className="flex gap-2 items-center bg-grayA-3 rounded-md p-1.5 transition-all duration-200 ease-in-out transform animate-in fade-in slide-in-from-bottom-2"
        key={slug}
      >
        <HandHoldingKey size="sm-regular" className="text-grayA-11" />
        <span className="text-gray-11 text-xs font-mono">{slug}</span>
      </div>
    ));
  }, [displaySlugs]);

  if (displaySlugs.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <div className="flex gap-2 items-center transition-all duration-300 ease-in-out">
        <div className="font-medium text-sm text-gray-12">Granted Access</div>
        <div
          className={`
            rounded-full border bg-grayA-3 border-grayA-3 w-[22px] h-[18px] 
            flex items-center justify-center font-medium text-[11px] text-grayA-12
            transition-all duration-300 ease-in-out transform
            ${isLoading ? "animate-pulse" : "animate-in zoom-in-50"}
          `}
        >
          {isLoading ? "..." : displayCount}
        </div>
      </div>

      <div className="h-[1px] bg-grayA-3 w-full transition-opacity duration-200" />

      <div
        className={`
          flex flex-wrap gap-1 items-center min-h-[2rem]
          transition-all duration-300 ease-in-out
          ${isLoading ? "opacity-50" : "opacity-100"}
        `}
      >
        {isLoading ? (
          <div className="flex gap-1">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-7 w-20 bg-grayA-4 rounded-md animate-pulse"
                style={{ animationDelay: `${i * 100}ms` }}
              />
            ))}
          </div>
        ) : displaySlugs.length > 0 ? (
          memoizedSlugs
        ) : (
          <div className="text-grayA-9 text-xs italic py-2 animate-in fade-in">
            No permissions selected
          </div>
        )}
      </div>
    </div>
  );
};

export const KeyRbacDialog = ({
  existingKey,
  triggerButton,
  isOpen: externalIsOpen,
  onClose: externalOnClose,
}: KeyRbacDialogProps) => {
  const { data, isLoading } = trpc.key.connectedRolesAndPerms.useQuery({
    keyId: existingKey.id,
  });

  const [internalIsOpen, setInternalIsOpen] = useState(false);

  const isDialogOpen = externalIsOpen !== undefined ? externalIsOpen : internalIsOpen;
  const setIsDialogOpen =
    externalOnClose !== undefined
      ? (open: boolean) => !open && externalOnClose()
      : setInternalIsOpen;

  const storageKey = `${FORM_STORAGE_KEY}_${existingKey.id}`;
  const methods = usePersistedForm<FormValues>(
    storageKey,
    {
      resolver: zodResolver(updateKeyRbacSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getDefaultValues(existingKey),
    },
    "memory",
  );

  const {
    formState: { isValid },
    handleSubmit,
    reset,
    clearPersistedData,
    saveCurrentValues,
    loadSavedValues,
    control,
    watch,
  } = methods;

  // Watch form values with debouncing for performance
  const watchedRoleIds = watch("roleIds");
  const watchedPermissionIds = watch("permissionIds");

  // Debounced slugs fetch - only enabled when dialog is open
  const { data: dataSlugs, isLoading: isSlugsLoading } = useFetchPermissionSlugs(
    watchedRoleIds,
    watchedPermissionIds,
    isDialogOpen, // Only fetch when dialog is open
  );

  // Only reset when dialog opens AND data is loaded
  useEffect(() => {
    if (!isDialogOpen || isLoading) {
      return;
    }

    const loadData = async () => {
      const hasSavedData = await loadSavedValues();
      if (!hasSavedData) {
        // Reset with API data when available
        const defaultValues = getDefaultValues(existingKey, data);
        reset(defaultValues);
      }
    };

    loadData();
  }, [existingKey, reset, loadSavedValues, isDialogOpen, data, isLoading]);

  const updateKeyRbacMutation = useUpdateKeyRbac(() => {
    clearPersistedData();
    reset(getDefaultValues(existingKey));
    setIsDialogOpen(false);
  });

  const onSubmit = async (data: FormValues) => {
    updateKeyRbacMutation.mutate(data);
  };

  const handleDialogToggle = (open: boolean) => {
    if (!open) {
      saveCurrentValues();
    }
    setIsDialogOpen(open);
  };

  // Don't render form until we have the key data
  if (isLoading && isDialogOpen) {
    return (
      <DialogContainer
        title={DIALOG_CONFIG.title}
        subTitle="Loading..."
        isOpen={isDialogOpen}
        onOpenChange={handleDialogToggle}
      >
        <div className="p-4 animate-pulse">Loading key data...</div>
      </DialogContainer>
    );
  }

  const defaultTrigger = (
    <NavbarActionButton title={DIALOG_CONFIG.triggerTitle} onClick={() => setIsDialogOpen(true)}>
      <PenWriting3 />
      {DIALOG_CONFIG.triggerTitle}
    </NavbarActionButton>
  );

  return (
    <>
      {triggerButton && <Navbar.Actions>{defaultTrigger}</Navbar.Actions>}
      <FormProvider {...methods}>
        <form id={`key-rbac-form-${existingKey.id}`} onSubmit={handleSubmit(onSubmit)}>
          <Controller
            name="keyId"
            control={control}
            render={({ field }) => <input type="hidden" {...field} />}
          />
          <DialogContainer
            title={DIALOG_CONFIG.title}
            subTitle={DIALOG_CONFIG.subtitle}
            isOpen={isDialogOpen}
            onOpenChange={handleDialogToggle}
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="submit"
                  form={`key-rbac-form-${existingKey.id}`}
                  variant="primary"
                  size="xlg"
                  className="w-full rounded-lg transition-all duration-200"
                  disabled={!isValid || updateKeyRbacMutation.isLoading}
                  loading={updateKeyRbacMutation.isLoading}
                >
                  {DIALOG_CONFIG.buttonText}
                </Button>
                <div className="text-gray-9 text-xs">{DIALOG_CONFIG.footerText}</div>
              </div>
            }
          >
            <div className="space-y-5 px-2 py-1">
              <KeyInfo
                keyDetails={{
                  id: existingKey.id,
                  name: existingKey.name ?? null,
                }}
              />
              <div className="py-1 my-2">
                <div className="h-[1px] bg-grayA-3 w-full" />
              </div>
              <Controller
                name="roleIds"
                control={control}
                render={({ field, fieldState }) => (
                  <RoleField
                    value={field.value ?? []}
                    onChange={field.onChange}
                    error={fieldState.error?.message}
                    keyId={existingKey.id}
                    assignedRoleDetails={data?.roles ?? []}
                  />
                )}
              />
              <Controller
                name="permissionIds"
                control={control}
                render={({ field, fieldState }) => (
                  <PermissionField
                    value={field.value ?? []}
                    onChange={field.onChange}
                    error={fieldState.error?.message}
                    assignedPermsDetails={data?.permissions ?? []}
                  />
                )}
              />

              <GrantedAccess
                slugs={dataSlugs?.slugs}
                totalCount={dataSlugs?.totalCount}
                isLoading={isSlugsLoading}
              />
            </div>
          </DialogContainer>
        </form>
      </FormProvider>
    </>
  );
};
