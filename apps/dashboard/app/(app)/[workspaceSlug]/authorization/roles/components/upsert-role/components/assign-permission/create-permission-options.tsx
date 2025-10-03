import { StatusBadge } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/settings/components/status-badge";
import { Lock, Page2 } from "@unkey/icons";
import { Badge, Button, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";

type Permission = {
  id: string;
  name: string;
  description: string | null;
  slug: string;
  roles: {
    id: string;
    name: string;
  }[];
};

type PermissionSelectorProps = {
  permissions: Permission[];
  hasNextPage?: boolean;
  isFetchingNextPage: boolean;
  roleId?: string;
  loadMore: () => void;
};

export function createPermissionOptions({
  permissions,
  hasNextPage,
  isFetchingNextPage,
  roleId,
  loadMore,
}: PermissionSelectorProps) {
  const options = permissions.map((permission) => ({
    label: (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex w-full text-accent-8 text-xs gap-4 py-0.5 items-center group flex-row">
              <div className="border rounded-full flex items-center justify-center border-grayA-6 size-5">
                <Page2 iconsize="sm-regular" className="text-grayA-11" />
              </div>
              <div className="flex gap-1 flex-col truncate min-w-0 flex-1">
                <div className="flex gap-2 items-center min-w-0">
                  <div className="flex flex-col gap-0.5 truncate min-w-0">
                    <div className="flex items-center  gap-2">
                      <span className="font-medium text-accent-12 text-left truncate">
                        {permission.name}
                      </span>
                      {permission.roles.find((item) => item.id === roleId) && (
                        <StatusBadge
                          variant="locked"
                          text="Already assigned"
                          icon={<Lock iconsize="sm-thin" />}
                        />
                      )}
                    </div>
                    <span className="text-accent-10 text-xs font-mono truncate">
                      {permission.slug}
                    </span>
                  </div>
                </div>
                {permission.description && (
                  <span className="text-accent-9 text-xs leading-tight">
                    {permission.description}
                  </span>
                )}
              </div>
            </div>
          </TooltipTrigger>
          <TooltipContent
            side="right"
            align="start"
            sideOffset={30}
            className="drop-shadow-2xl border border-grayA-4 overflow-hidden rounded-[10px] p-0 bg-white dark:bg-black w-[320px] z-[100]"
          >
            <div className="flex flex-col h-full">
              {/* Header */}
              <div className="px-4 py-2 border-b border-grayA-4 text-gray-10 text-xs font-medium bg-grayA-2">
                Permission Details
              </div>
              {/* Content */}
              <div className="px-4 py-3 space-y-3">
                <div>
                  <div className="text-xs font-medium text-gray-11 mb-1">Permission ID</div>
                  <div className="text-xs text-gray-12 font-mono break-all">{permission.id}</div>
                </div>
                <div>
                  <div className="text-xs font-medium text-gray-11 mb-1">Name</div>
                  <div className="text-xs text-gray-12">{permission.name}</div>
                </div>
                <div>
                  <div className="text-xs font-medium text-gray-11 mb-1">Slug</div>
                  <div className="text-xs text-gray-12 font-mono">{permission.slug}</div>
                </div>
                {permission.description && (
                  <div>
                    <div className="text-xs font-medium text-gray-11 mb-1">Description</div>
                    <div className="text-xs text-gray-12">{permission.description}</div>
                  </div>
                )}
                {permission.roles.length > 0 && (
                  <div>
                    <div className="text-xs font-medium text-gray-11 mb-2">Roles</div>
                    <div className="flex flex-wrap gap-1">
                      {permission.roles.map((role) => (
                        <Badge key={role.id} variant="secondary" className="text-xs">
                          {role.name}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ),
    selectedLabel: (
      <div className="flex w-full text-accent-8 text-xs gap-1.5 py-0.5 items-center">
        <div className="flex items-center justify-center gap-2">
          <div className="border rounded-full flex items-center justify-center border-grayA-6 size-5">
            <Page2 iconsize="sm-regular" className="text-grayA-11" />
          </div>
          <span className="text-accent-12 font-medium text-xs w-[120px] truncate text-left">
            {permission.id.length > 15
              ? `${permission.id.slice(0, 8)}...${permission.id.slice(-4)}`
              : permission.id}
          </span>
        </div>
        <span className="w-[200px] truncate text-accent-8 text-left">{permission.name}</span>
      </div>
    ),
    value: permission.id,
    searchValue: `${permission.id} ${permission.name} ${permission.slug} ${
      permission.description || ""
    }`.trim(),
  }));

  if (hasNextPage) {
    options.push({
      label: (
        <Button
          type="button"
          variant="ghost"
          loading={isFetchingNextPage}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            loadMore();
          }}
          className="text-xs text-accent-12 px-2 py-0.5 hover:bg-grayA-3 rounded w-full bg-transparent hover:bg-transparent focus:ring-0 font-medium"
        >
          Load more...
        </Button>
      ),
      value: "__load_more__",
      selectedLabel: <></>,
      searchValue: "",
    });
  }

  return options;
}
