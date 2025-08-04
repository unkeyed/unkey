"use client";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTrigger,
} from "@/components/ui/sheet";

import { ScrollArea } from "@/components/ui/scroll-area";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button } from "@unkey/ui";

import { useCallback, useMemo, useRef, useState } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";

// Type definitions for permission structure
type PermissionItem = {
  description: string;
  permission: string; // Using string instead of UnkeyPermission to handle literal types
};
type PermissionCategory = Record<string, PermissionItem>;
type PermissionList = Record<string, PermissionCategory>;
import { PermissionContentList } from "./permission-list";
import { SearchPermissions } from "./search-permissions";

type PermissionSheetProps = {
  children: React.ReactNode;
  apis: {
    id: string;
    name: string;
  }[];
  selectedPermissions: UnkeyPermission[];
  onChange?: (permissions: UnkeyPermission[]) => void;
  loadMore?: () => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  editMode?: boolean;
  isLoading?: boolean;
};

export const PermissionSheet = ({
  children,
  apis,
  selectedPermissions,
  onChange,
  loadMore,
  hasNextPage,
  isFetchingNextPage,
  editMode = false,
  isLoading = false,
}: PermissionSheetProps) => {
  const [open, setOpen] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const [searchValue, setSearchValue] = useState<string | undefined>(undefined);

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setIsProcessing(true);
    if (e.target.value === "") {
      setSearchValue(undefined);
    } else {
      setSearchValue(e.target.value);
    }
    setIsProcessing(false);
  };

  const handleOpenChange = useCallback(
    (open: boolean) => {
      // Only allow opening if APIs are loaded or we have selected permissions (edit mode)
      if (open && isLoading && !editMode && selectedPermissions.length === 0) {
        return;
      }

      setOpen(open);
    },
    [isLoading, editMode, selectedPermissions.length],
  );

  const handleApiPermissionChange = (apiId: string, permissions: UnkeyPermission[]) => {
    // Get all current permissions and update the specific API's permissions
    const currentApiPerms = selectedPermissions.filter((permission) => {
      // Check if this permission belongs to the specific API
      const apiPermsList = Object.values(apiPermissions(apiId)).flatMap((category) =>
        Object.values(category).map((item) => item.permission),
      );
      return apiPermsList.includes(permission);
    });

    // Get workspace permissions
    const workspacePerms = selectedPermissions.filter((permission) => {
      const workspacePermsList = Object.values(workspacePermissions).flatMap((category) =>
        Object.values(category).map((item) => item.permission),
      );
      return workspacePermsList.includes(permission);
    });

    // Get other API permissions
    const otherApiPerms = selectedPermissions.filter((permission) => {
      const workspacePermsList = Object.values(workspacePermissions).flatMap((category) =>
        Object.values(category).map((item) => item.permission),
      );
      if (workspacePermsList.includes(permission)) return false;

      for (const api of apis) {
        if (api.id === apiId) continue;
        const apiPermsList = Object.values(apiPermissions(api.id)).flatMap((category) =>
          Object.values(category).map((item) => item.permission),
        );
        if (apiPermsList.includes(permission)) return true;
      }
      return false;
    });

    // Combine all permissions
    const allPermissions = [...workspacePerms, ...otherApiPerms, ...permissions];
    onChange?.(allPermissions);
  };

  const handleWorkspacePermissionChange = (permissions: UnkeyPermission[]) => {
    // Get all current API permissions
    const apiPerms = selectedPermissions.filter((permission) => {
      const workspacePermsList = Object.values(workspacePermissions).flatMap((category) =>
        Object.values(category).map((item) => item.permission),
      );
      return !workspacePermsList.includes(permission);
    });

    // Combine workspace and API permissions
    const allPermissions = [...permissions, ...apiPerms];
    onChange?.(allPermissions);
  };

  // Helper function to check if permission list has any matching results
  const hasPermissionResults = (permissionList: PermissionList, searchValue?: string) => {
    if (!searchValue || searchValue.trim() === "") {
      // If no search, check if any categories have permissions
      return Object.keys(permissionList).some(
        (category) => Object.keys(permissionList[category]).length > 0,
      );
    }

    // If searching, check if any permission names match
    const searchLower = searchValue.toLowerCase();
    return Object.values(permissionList).some((category: PermissionCategory) =>
      Object.keys(category).some((permissionName) =>
        permissionName.toLowerCase().includes(searchLower),
      ),
    );
  };

  // Check if all permission lists are empty after filtering
  const hasNoResults = useMemo(() => {
    // Check workspace permissions
    const workspaceHasResults = hasPermissionResults(workspacePermissions, searchValue);

    // Check API permissions
    const anyApiHasResults = apis.some((api) => {
      const apiPerms = apiPermissions(api.id);
      return hasPermissionResults(apiPerms, searchValue);
    });

    return !workspaceHasResults && (apis.length === 0 || !anyApiHasResults);
  }, [searchValue, apis]);

  return (
    <Sheet open={open} onOpenChange={handleOpenChange} modal={true}>
      <SheetTrigger asChild>{children}</SheetTrigger>
      <SheetContent
        disableClose={true}
        className="flex flex-col p-0 m-0 h-full gap-0 border-l border-l-gray-4 w-[420px]"
        side="right"
        overlay="transparent"
      >
        <SheetHeader className="flex flex-row w-full border-b border-gray-4 gap-2">
          <SearchPermissions
            isProcessing={isProcessing}
            search={searchValue}
            inputRef={inputRef}
            onChange={handleSearchChange}
          />
        </SheetHeader>
        <SheetDescription className="w-full h-full">
          <div className="flex flex-col h-full">
            <div
              className={`flex flex-col ${
                hasNextPage ? "max-h-[calc(100%-80px)]" : "max-h-[calc(100%-40px)]"
              }`}
            >
              <ScrollArea className="flex flex-col h-full pt-2">
                <div className="flex flex-col pt-0 mt-0 gap-1 pb-6">
                  {hasNoResults ? (
                    <p className="text-sm text-gray-10 ml-6 py-auto mt-1.5">No results found</p>
                  ) : (
                    <>
                      {/* Workspace Permissions */}
                      <PermissionContentList
                        selected={selectedPermissions}
                        searchValue={searchValue}
                        key="workspace"
                        type="workspace"
                        onPermissionChange={(permissions) =>
                          handleWorkspacePermissionChange(permissions)
                        }
                      />
                      {/* From APIs */}
                      {apis.length > 0 && (
                        <p className="text-sm text-gray-10 ml-6 py-auto mb-2">From APIs</p>
                      )}
                      {apis.map((api) => (
                        <PermissionContentList
                          selected={selectedPermissions}
                          searchValue={searchValue}
                          key={api.id}
                          type="api"
                          api={api}
                          onPermissionChange={(permissions) =>
                            handleApiPermissionChange(api.id, permissions)
                          }
                        />
                      ))}
                    </>
                  )}
                </div>
              </ScrollArea>
            </div>
            <div className="absolute bottom-2 right-0 max-h-10 w-full">
              {hasNextPage ? (
                <div className="absolute bottom-0 right-0 w-full h-fit py-4">
                  <div className="flex flex-row justify-end items-center">
                    <Button
                      className="mx-auto w-18 rounded-lg"
                      size="sm"
                      onClick={loadMore}
                      disabled={!hasNextPage}
                      loading={isFetchingNextPage}
                    >
                      Load More
                    </Button>
                  </div>
                </div>
              ) : undefined}
            </div>
          </div>
        </SheetDescription>
      </SheetContent>
    </Sheet>
  );
};
