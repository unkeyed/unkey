"use client";

import { cn } from "@/lib/utils";
import type { UnkeyPermission } from "@unkey/rbac";
import { Dialog, DialogContent, DialogTitle } from "@unkey/ui";
import { SimpleChainVariant } from "./urn-permission-variants/simple-chain";
import type {
  ApiItem,
  PermissionResourceSuggestions,
  ScopedItem,
} from "./urn-permission-variants/types";

type UrnPermissionSheetProps = {
  workspaceId: string;
  apis: ApiItem[];
  projects: ScopedItem[];
  permissionResources?: PermissionResourceSuggestions;
  selectedPermissions: UnkeyPermission[];
  onChange: (permissions: UnkeyPermission[]) => void;
  loadMore?: () => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function UrnPermissionSheet({
  workspaceId,
  apis,
  projects,
  selectedPermissions,
  permissionResources,
  onChange,
  loadMore,
  hasNextPage,
  isFetchingNextPage,
  open,
  onOpenChange,
}: UrnPermissionSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={cn(
          "drop-shadow-2xl border-grayA-4 overflow-hidden rounded-2xl! p-0 gap-0 max-w-none max-h-[88vh]",
          "w-[min(1160px,calc(100vw-48px))]",
        )}
      >
        <div className="border-b border-gray-4 px-6 py-4 bg-white dark:bg-black">
          <DialogTitle className="text-base font-medium text-gray-12">Add permissions</DialogTitle>
          <p className="text-[13px] text-gray-10 mt-1">
            Choose an action, then build the resource path this key can access.
          </p>
        </div>
        <div className="max-h-[calc(88vh-73px)] overflow-auto bg-grayA-2 p-5">
          <SimpleChainVariant
            workspaceId={workspaceId}
            apis={apis}
            projects={projects}
            permissionResources={permissionResources}
            selectedPermissions={selectedPermissions}
            onChange={onChange}
            loadMore={loadMore}
            hasNextPage={hasNextPage}
            isFetchingNextPage={isFetchingNextPage}
            onClose={() => onOpenChange(false)}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
