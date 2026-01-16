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
      <div className="flex flex-col gap-1 mt-[16px] px-4">
        <div className="border bg-gray-2 border-gray-4 rounded-[10px]">
          <div className="text-gray-11 text-[12px] leading-6 px-[14px] py-1.5 font-sans">Roles</div>
          <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2">
            <span className="text-xs text-gray-11">No roles assigned</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 mt-[16px] px-4">
      <div className="border bg-gray-2 border-gray-4 rounded-[10px]">
        <div className="text-gray-11 text-[12px] leading-6 px-[14px] py-1.5 font-sans">
          Roles ({roles.length})
        </div>
        <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2 space-y-2">
          {roles.map((role) => (
            <div
              key={role.name}
              className="relative group p-2 rounded-md border border-gray-4 bg-gray-2 hover:border-gray-5 transition-colors"
            >
              <div className="text-accent-12 font-medium text-xs">{role.name}</div>
              {role.description ? (
                <div className="text-accent-10 mt-1 text-xs">{role.description}</div>
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
      <div className="flex flex-col gap-1 mt-[16px] px-4">
        <div className="border bg-gray-2 border-gray-4 rounded-[10px]">
          <div className="text-gray-11 text-[12px] leading-6 px-[14px] py-1.5 font-sans">
            Permissions
          </div>
          <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2">
            <span className="text-xs text-gray-11">No permissions assigned</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 mt-[16px] px-4">
      <div className="border bg-gray-2 border-gray-4 rounded-[10px] relative">
        <div className="text-gray-11 text-[12px] leading-6 px-[14px] py-1.5 font-sans flex items-center justify-between">
          <span>Permissions ({permissions.length})</span>
          {permissions.length > 1 && (
            <CopyButton
              value={permissions
                .map(
                  (permission) =>
                    `${permission.name}${permission.description ? `\n${permission.description}` : ""
                    }`,
                )
                .join("\n\n")}
              shape="square"
              variant="outline"
              className="h-6 w-6 rounded-sm bg-gray-2 hover:bg-gray-2"
              aria-label="Copy all permissions"
            />
          )}
        </div>
        <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2 space-y-2">
          {permissions.map((permission) => (
            <div
              key={permission.name}
              className="relative group p-2 rounded-md border border-gray-4 bg-gray-2 hover:border-gray-5 transition-colors"
            >
              <div className="text-accent-12 font-medium text-xs">{permission.name}</div>
              {permission.description ? (
                <div className="text-accent-10 mt-1 text-xs">{permission.description}</div>
              ) : (
                <div className="text-gray-11 mt-1 text-xs italic">No description</div>
              )}
              <CopyButton
                value={`${permission.name}${permission.description ? `\n${permission.description}` : ""
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
