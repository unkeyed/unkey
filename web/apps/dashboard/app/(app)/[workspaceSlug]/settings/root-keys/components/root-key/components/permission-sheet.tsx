"use client";

import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button } from "@unkey/ui";
import { useCallback, useRef } from "react";
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
  const scrollRef = useRef<HTMLDivElement>(null);

  const handleWheel = useCallback((e: React.WheelEvent<HTMLDivElement>) => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop += e.deltaY;
    }
  }, []);

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
      <SheetContent
        disableClose={false}
        className="flex flex-col p-0 m-0 h-full gap-0 border-l border-l-gray-4 w-[420px] bg-gray-1 dark:bg-black overflow-hidden"
        side="right"
        overlay="transparent"
      >
        <SheetHeader className="flex flex-row min-w-full border-b border-gray-4 gap-2 shrink-0">
          <SheetTitle className="sr-only">Select Permissions</SheetTitle>
          <SearchPermissions
            isProcessing={isProcessing}
            search={searchValue}
            inputRef={inputRef}
            onChange={handleSearchChange}
          />
        </SheetHeader>
        <div
          ref={scrollRef}
          className="flex-1 min-h-0 overflow-y-auto overscroll-contain"
          onWheel={handleWheel}
        >
          <div className="flex flex-col gap-1 pt-2 pb-6">
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
        </div>
        {hasNextPage && (
          <div className="shrink-0 border-t border-gray-4 py-4">
            <div className="flex justify-center">
              <Button
                className="rounded-lg"
                size="sm"
                onClick={() => loadMore?.()}
                disabled={!hasNextPage || !loadMore}
                loading={isFetchingNextPage}
              >
                {ROOT_KEY_MESSAGES.UI.LOAD_MORE}
              </Button>
            </div>
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
};
