import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Clone } from "@unkey/icons";
import { Button } from "@unkey/ui";

type Role = {
  name: string;
  description: string | null;
};

type RolesSectionProps = {
  roles: Role[];
};

export const RolesSection: React.FC<RolesSectionProps> = ({ roles }) => {
  const handleCopy = (role: Role) => {
    const content = `${role.name}${role.description ? `\n${role.description}` : ""}`;
    navigator.clipboard
      .writeText(content)
      .then(() => {
        toast.success(`Role ${role.name} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  if (!roles || roles.length === 0) {
    return (
      <div className="flex flex-col gap-1 mt-[16px]">
        <div className="flex justify-between items-center">
          <span className="text-[13px] text-accent-9 font-sans">Roles</span>
        </div>
        <Card className="bg-gray-2 border-gray-4 rounded-lg">
          <CardContent className="py-2 px-3 text-xs">
            <span className="text-accent-9">No roles assigned</span>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <span className="text-[13px] text-accent-9 font-sans">Roles ({roles.length})</span>
      <div className="space-y-2">
        {roles.map((role) => (
          <Card
            key={role.name}
            className="bg-gray-2 border-gray-4 rounded-lg hover:border-gray-5 transition-colors"
          >
            <CardContent className="py-3 px-4 text-xs relative group">
              <div className="flex items-center">
                <span className="text-accent-12 font-medium">{role.name}</span>
              </div>
              {role.description ? (
                <div className="text-accent-10 mt-2 text-xs">{role.description}</div>
              ) : (
                <div className="text-accent-9 mt-2 text-xs italic">No description</div>
              )}
              <Button
                shape="square"
                onClick={() => handleCopy(role)}
                variant="outline"
                className="absolute bottom-2 right-3 opacity-0 group-hover:opacity-100 transition-opacity rounded-sm"
                aria-label={`Copy ${role.name}`}
              >
                <Clone />
              </Button>
            </CardContent>
          </Card>
        ))}
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

export const PermissionsSection: React.FC<PermissionsSectionProps> = ({ permissions }) => {
  const handleCopy = (permission: Permission) => {
    const content = `${permission.name}${
      permission.description ? `\n${permission.description}` : ""
    }`;
    navigator.clipboard
      .writeText(content)
      .then(() => {
        toast.success(`Permission ${permission.name} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  const handleCopyAll = () => {
    const content = permissions
      .map(
        (permission) =>
          `${permission.name}${permission.description ? `\n${permission.description}` : ""}`,
      )
      .join("\n\n");

    navigator.clipboard
      .writeText(content)
      .then(() => {
        toast.success("All permissions copied to clipboard");
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  if (!permissions || permissions.length === 0) {
    return (
      <div className="flex flex-col gap-1 mt-[16px]">
        <div className="flex justify-between items-center">
          <span className="text-[13px] text-accent-9 font-sans">Permissions</span>
        </div>
        <Card className="bg-gray-2 border-gray-4 rounded-lg">
          <CardContent className="py-2 px-3 text-xs">
            <span className="text-accent-9">No permissions assigned</span>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">
          Permissions ({permissions.length})
        </span>
        {permissions.length > 1 && (
          <Button
            shape="square"
            onClick={handleCopyAll}
            variant="outline"
            className="h-6 w-6 rounded-sm"
            aria-label="Copy all permissions"
          >
            <Clone />
          </Button>
        )}
      </div>
      <div className="space-y-2">
        {permissions.map((permission) => (
          <Card
            key={permission.name}
            className="bg-gray-2 border-gray-4 rounded-lg hover:border-gray-5 transition-colors"
          >
            <CardContent className="py-3 px-4 text-xs relative group">
              <div className="flex items-center gap-2">
                <span className="text-accent-12 font-medium">{permission.name}</span>
              </div>
              {permission.description ? (
                <div className="text-accent-10 mt-2 text-xs">{permission.description}</div>
              ) : (
                <div className="text-accent-9 mt-2 text-xs italic">No description</div>
              )}
              <Button
                shape="square"
                onClick={() => handleCopy(permission)}
                variant="outline"
                className="absolute bottom-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity rounded-sm"
                aria-label={`Copy ${permission.name}`}
              >
                <Clone />
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
};
