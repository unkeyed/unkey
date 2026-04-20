"use client";

import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetOverlay,
  SheetPortal,
  SheetTitle,
} from "@/components/ui/sheet";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button } from "@unkey/ui";
import { useRef } from "react";
import { ROOT_KEY_MESSAGES } from "../constants";
import { usePermissionSheet } from "../hooks/use-permission-sheet";
import { PermissionContentList } from "./permission-list";
import { SearchPermissions } from "./search-permissions";

type PermissionSheetProps = {
  apis: { id: string; name: string }[];
  selectedPermissions: UnkeyPermission[];
  onChange: (permissions: UnkeyPermission[]) => void;
  loadMore?: () => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  editMode?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export const PermissionSheet = ({
  apis,
  selectedPermissions,
  onChange,
  loadMore,
  hasNextPage,
  isFetchingNextPage,
  editMode = false,
  open,
  onOpenChange,
}: PermissionSheetProps) => {
  const inputRef = useRef<HTMLInputElement>(null);

  const {
    searchValue,
    isProcessing,
    hasNoResults,
    handleSearchChange,
    handleApiPermissionChange,
    handleWorkspacePermissionChange,
  } = usePermissionSheet({
    apis,
    selectedPermissions,
    onChange,
    editMode,
  });

  return (
    <Sheet modal={true} open={open} onOpenChange={onOpenChange}>
      <SheetPortal>
        <SheetOverlay className="bg-black/30 backdrop-blur-xs" />
        <SheetContent
          disableClose={false}
          className="flex flex-col overflow-hidden p-0 m-0 h-full gap-0 border-l border-l-gray-4 w-[420px] bg-gray-1 dark:bg-black"
          side="right"
          overlay="transparent"
        >
          <SheetHeader className="flex flex-row shrink-0 min-w-full border-b border-gray-4 gap-2">
            <SheetTitle className="sr-only">Select Permissions</SheetTitle>
            <SearchPermissions
              isProcessing={isProcessing}
              search={searchValue}
              inputRef={inputRef}
              onChange={handleSearchChange}
            />
          </SheetHeader>
          <div className="flex flex-col flex-1 min-h-0 overflow-hidden">
            <ScrollArea className="flex-1 min-h-0 pt-2">
              <div className="flex flex-col pt-0 mt-0 gap-1 pb-6">
                {hasNoResults ? (
                  <p className="text-sm text-gray-10 ml-6 py-1.5 mt-1.5">
                    {ROOT_KEY_MESSAGES.UI.NO_RESULTS}
                  </p>
                ) : (
                  <>
                    <PermissionContentList
                      selected={selectedPermissions}
                      searchValue={searchValue}
                      key="workspace"
                      type="workspace"
                      onPermissionChange={handleWorkspacePermissionChange}
                    />
                    {apis.length > 0 && (
                      <p className="text-sm text-gray-10 ml-6 py-1.5 mb-2">
                        {ROOT_KEY_MESSAGES.UI.FROM_APIS}
                      </p>
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
          {hasNextPage && (
            <div className="shrink-0 bg-background border-t border-gray-4">
              <div className="w-full py-4">
                <div className="flex flex-row justify-center items-center">
                  <Button
                    className="mx-auto rounded-lg"
                    size="sm"
                    onClick={() => loadMore?.()}
                    disabled={!hasNextPage || !loadMore}
                    loading={isFetchingNextPage}
                  >
                    {ROOT_KEY_MESSAGES.UI.LOAD_MORE}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </SheetContent>
      </SheetPortal>
    </Sheet>
  );
};
