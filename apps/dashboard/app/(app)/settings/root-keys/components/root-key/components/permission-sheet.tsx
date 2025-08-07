"use client";

import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTrigger,
} from "@/components/ui/sheet";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button } from "@unkey/ui";
import { useRef } from "react";
import { ROOT_KEY_MESSAGES } from "../constants";
import { usePermissionSheet } from "../hooks/use-permission-sheet";
import { PermissionContentList } from "./permission-list";
import { SearchPermissions } from "./search-permissions";

type PermissionSheetProps = {
  children: React.ReactNode;
  apis: { id: string; name: string }[];
  selectedPermissions: UnkeyPermission[];
  onChange?: (permissions: UnkeyPermission[]) => void;
  loadMore?: () => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  editMode?: boolean;
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
    <Sheet modal={true}>
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
                    <p className="text-sm text-gray-10 ml-6 py-auto mt-1.5">
                      {ROOT_KEY_MESSAGES.UI.NO_RESULTS}
                    </p>
                  ) : (
                    <>
                      {/* Workspace Permissions */}
                      <PermissionContentList
                        selected={selectedPermissions}
                        searchValue={searchValue}
                        key="workspace"
                        type="workspace"
                        onPermissionChange={handleWorkspacePermissionChange}
                      />
                      {/* From APIs */}
                      {apis.length > 0 && (
                        <p className="text-sm text-gray-10 ml-6 py-auto mb-2">
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
                      {ROOT_KEY_MESSAGES.UI.LOAD_MORE}
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
