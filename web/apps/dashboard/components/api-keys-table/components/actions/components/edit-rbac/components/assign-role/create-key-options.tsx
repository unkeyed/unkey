import { StatusBadge } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/settings/components/status-badge";
import type { Role } from "@unkey/api/models/components";
import { IconLockOutline12, IconTagOutline12 } from "@unkey/icons";
import { Badge, Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";

type RoleSelectorProps = {
  roles: Role[];
  hasNextPage?: boolean;
  isFetchingNextPage: boolean;
  previouslySelectedRoleNames: string[];
  loadMore: () => void;
};

export function createRoleOptions({
  roles,
  hasNextPage,
  isFetchingNextPage,
  previouslySelectedRoleNames,
  loadMore,
}: RoleSelectorProps) {
  const options = roles.map((role) => ({
    label: (
      <HoverCard>
        <HoverCardTrigger
          render={
            <div className="flex w-full text-gray-8 text-xs gap-4 py-0.5 items-center group flex-row">
              <div className="border rounded-full flex items-center justify-center size-5">
                <IconTagOutline12 className="text-grayA-11" />
              </div>
              <div className="flex gap-1 flex-col truncate">
                <div className="flex gap-2 items-center">
                  <span className="font-medium text-gray-12 text-left">{role.name}</span>
                  {previouslySelectedRoleNames.includes(role.name) && (
                    <StatusBadge
                      variant="locked"
                      text="Already assigned"
                      icon={<IconLockOutline12 />}
                    />
                  )}
                </div>
                <span className="text-gray-9 text-xs truncate">
                  {role.description || `${role.permissions?.length ?? 0} permissions`}
                </span>
              </div>
            </div>
          }
        />
        <HoverCardContent
          side="right"
          align="start"
          sideOffset={30}
          className="shadow-floating overflow-hidden rounded-xl p-0 w-80 z-100"
        >
          <div className="flex flex-col h-full">
            {/* Header */}
            <div className="px-4 py-2 border-b text-gray-10 text-xs font-medium bg-grayA-2">
              Role Details
            </div>
            {/* Content */}
            <div className="flex flex-col px-4 py-3 gap-3">
              <div>
                <div className="text-xs font-medium text-gray-11 mb-1">Role Name</div>
                <div className="text-xs text-gray-12 break-all">{role.name}</div>
              </div>
              {role.description && (
                <div>
                  <div className="text-xs font-medium text-gray-11 mb-1">Description</div>
                  <div className="text-xs text-gray-12">{role.description}</div>
                </div>
              )}
              <div>
                <div className="text-xs font-medium text-gray-11 mb-1">Role ID</div>
                <div className="text-xs text-gray-12 font-mono break-all">{role.id}</div>
              </div>
              {(role.permissions?.length ?? 0) > 0 && (
                <div>
                  <div className="text-xs font-medium text-gray-11 mb-2">Permissions</div>
                  <div className="flex flex-wrap gap-1">
                    {role.permissions?.map((permission) => (
                      <Badge key={permission.id} variant="secondary" className="text-xs">
                        {permission.name}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </HoverCardContent>
      </HoverCard>
    ),
    selectedLabel: (
      <div className="flex w-full text-gray-8 text-xs gap-1.5 py-0.5 items-center">
        <div className="flex items-center justify-center gap-2">
          <div className="border rounded-full flex items-center justify-center size-5">
            <IconTagOutline12 className="text-grayA-11" />
          </div>
          <span className="text-gray-12 font-medium text-xs w-[120px] truncate text-left">
            {role.name}
          </span>
        </div>
        <span className="w-[200px] truncate text-gray-8 text-left">
          {role.description || `${role.permissions?.length ?? 0} permissions`}
        </span>
      </div>
    ),
    value: role.name,
    searchValue: `${role.name} ${role.description || ""} ${role.id}`.trim(),
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
          className="text-xs text-gray-12 px-2 py-0.5 hover:bg-grayA-3 rounded-sm w-full bg-transparent hover:bg-transparent focus:ring-0 font-medium"
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
