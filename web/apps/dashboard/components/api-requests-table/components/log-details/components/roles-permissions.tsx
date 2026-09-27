import { CopyButton } from "@unkey/ui";

type Role = {
  name: string;
  description: string | null;
};

type RolesSectionProps = {
  roles: Role[];
};

export const RolesSection: React.FC<RolesSectionProps> = ({ roles }) => {
  if (!roles || roles.length === 0) {
    return (
      <div className="flex flex-col gap-1 mt-4 px-4">
        <div className="border bg-raised rounded-xl">
          <div className="text-gray-11 text-xs leading-6 px-3.5 py-1.5 font-sans">Roles</div>
          <div className="border-t rounded-xl bg-raised px-4 py-2">
            <span className="text-xs text-gray-11">No roles assigned</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 mt-4 px-4">
      <div className="border bg-raised rounded-xl">
        <div className="text-gray-11 text-xs leading-6 px-4 py-1.5 font-sans">
          Roles ({roles.length})
        </div>
        <div className="flex flex-col border-t rounded-xl bg-raised px-3.5 py-2 gap-2">
          {roles.map((role) => (
            <div
              key={role.name}
              className="relative group p-2 rounded-md border bg-raised hover:border-strong transition-colors"
            >
              <div className="text-gray-12 font-medium text-xs">{role.name}</div>
              {role.description ? (
                <div className="text-gray-10 mt-1 text-xs">{role.description}</div>
              ) : (
                <div className="text-gray-11 mt-1 text-xs italic">No description</div>
              )}
              <CopyButton
                value={`${role.name}${role.description ? `\n${role.description}` : ""}`}
                shape="square"
                variant="outline"
                className="absolute bottom-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 bg-gray-2 hover:bg-gray-2 size-2"
                aria-label={`Copy ${role.name}`}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

type Permission = {
  name: string;
  description: string | null;
};

type PermissionsSectionProps = {
  permissions: Permission[];
};

export const PermissionsSection = ({ permissions }: PermissionsSectionProps) => {
  if (!permissions || permissions.length === 0) {
    return (
      <div className="flex flex-col gap-1 mt-4 px-4">
        <div className="border bg-raised rounded-xl">
          <div className="text-gray-11 text-xs leading-6 px-3.5 py-1.5 font-sans">Permissions</div>
          <div className="border-t rounded-xl bg-raised px-4 py-2">
            <span className="text-xs text-gray-11">No permissions assigned</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 mt-4 px-4">
      <div className="border bg-raised rounded-xl relative">
        <div className="text-gray-11 text-xs leading-6 px-3.5 py-1.5 font-sans flex items-center justify-between gap-2">
          <span>Permissions ({permissions.length})</span>
          {permissions.length > 1 && (
            <CopyButton
              value={permissions
                .map(
                  (permission) =>
                    `${permission.name}${
                      permission.description ? `\n${permission.description}` : ""
                    }`,
                )
                .join("\n\n")}
              shape="square"
              variant="outline"
              className="h-6 w-6 rounded-xs bg-gray-2 hover:bg-gray-2"
              aria-label="Copy all permissions"
            />
          )}
        </div>
        <div className="flex flex-col border-t rounded-xl bg-raised px-3.5 py-2 gap-2">
          {permissions.map((permission) => (
            <div
              key={permission.name}
              className="relative group p-2 rounded-md border bg-raised hover:border-strong transition-colors"
            >
              <div className="text-gray-12 font-medium text-xs">{permission.name}</div>
              {permission.description ? (
                <div className="text-gray-10 mt-1 text-xs">{permission.description}</div>
              ) : (
                <div className="text-gray-11 mt-1 text-xs italic">No description</div>
              )}
              <CopyButton
                value={`${permission.name}${
                  permission.description ? `\n${permission.description}` : ""
                }`}
                shape="square"
                variant="outline"
                className="absolute bottom-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 bg-gray-2 hover:bg-gray-2 size-2"
                aria-label={`Copy ${permission.name}`}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
