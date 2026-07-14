"use client";

import { PermissionField } from "@/components/api-keys-table/components/actions/components/edit-rbac/components/assign-permission/permissions-field";
import { RoleField } from "@/components/api-keys-table/components/actions/components/edit-rbac/components/assign-role/role-field";
import { GrantedAccess } from "@/components/api-keys-table/components/rbac/granted-access";
import { useFetchPermissionSlugs } from "@/components/api-keys-table/components/rbac/hooks/use-fetch-permission-slugs";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { FormValues } from "../create-key.schema";

export const RbacSetup = () => {
  const { control } = useFormContext<FormValues>();
  const roleNames = useWatch({ control, name: "roleNames", defaultValue: [] });
  const directPermissionSlugs = useWatch({
    control,
    name: "directPermissionSlugs",
    defaultValue: [],
  });

  const { data: permissionSlugs, isLoading: isPermissionSlugsLoading } = useFetchPermissionSlugs(
    roleNames,
    directPermissionSlugs,
  );

  return (
    <div className="flex flex-col gap-5">
      <Controller
        name="roleNames"
        control={control}
        render={({ field, fieldState }) => (
          <RoleField
            value={field.value ?? []}
            onChange={field.onChange}
            error={fieldState.error?.message}
            assignedRoleDetails={[]}
          />
        )}
      />
      <Controller
        name="directPermissionSlugs"
        control={control}
        render={({ field, fieldState }) => (
          <PermissionField
            value={field.value ?? []}
            onChange={field.onChange}
            error={fieldState.error?.message}
            assignedRoleDetails={[]}
            assignedPermsDetails={[]}
          />
        )}
      />
      <GrantedAccess
        slugs={permissionSlugs?.slugs}
        totalCount={permissionSlugs?.totalCount}
        isLoading={isPermissionSlugsLoading}
      />
    </div>
  );
};
