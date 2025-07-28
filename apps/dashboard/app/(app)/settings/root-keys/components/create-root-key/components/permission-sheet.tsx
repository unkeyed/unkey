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
import { useEffect, useRef, useState } from "react";
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
};
export const PermissionSheet = ({
  children,
  apis,
  selectedPermissions,
  onChange,
  loadMore,
  hasNextPage,
  isFetchingNextPage,
}: PermissionSheetProps) => {
  const [open, setOpen] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [search, setSearch] = useState<string | undefined>(undefined);
  const inputRef = useRef<HTMLInputElement>(null);
  const [workspacePermissions, setWorkspacePermissions] = useState<UnkeyPermission[]>([]);
  const [apiPermissions, setApiPermissions] = useState<Record<string, UnkeyPermission[]>>({});

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setIsProcessing(true);
    setSearch(e.target.value);
    // TODO: Implement actual search logic
    setIsProcessing(false);
  };

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
  };

  const handleApiPermissionChange = (apiId: string, permissions: UnkeyPermission[]) => {
    setApiPermissions((prev) => ({ ...prev, [apiId]: permissions }));
  };

  const handleWorkspacePermissionChange = (permissions: UnkeyPermission[]) => {
    setWorkspacePermissions(permissions);
  };

  // Aggregate all permissions and call onChange
  useEffect(() => {
    if (onChange) {
      const allApiPermissions = Object.values(apiPermissions).flat();
      onChange([...workspacePermissions, ...allApiPermissions]);
    }
  }, [workspacePermissions, apiPermissions, onChange]);

  return (
    <Sheet open={open} onOpenChange={handleOpenChange} modal={true}>
      <SheetTrigger asChild>{children}</SheetTrigger>
      <SheetContent
        className="flex flex-col p-0 m-0 h-full gap-0 border-l border-l-gray-4 min-w-[420px]"
        side="right"
        overlay="transparent"
      >
        <SheetHeader className="flex flex-row w-full border-b border-gray-4 gap-2">
          <SearchPermissions
            isProcessing={isProcessing}
            search={search}
            inputRef={inputRef}
            onChange={handleSearchChange}
          />
        </SheetHeader>
        <SheetDescription className="w-full h-full pt-2">
          <div className="flex flex-col h-full">
            <div
              className={`flex flex-col overflow-y-hidden ${hasNextPage ? "max-h-[calc(100%-110px)]" : "h-[calc(100%-40px)]"}`}
            >
              <ScrollArea className="flex flex-col h-full">
                <div className="flex flex-col pt-0 mt-0 gap-1 pb-4">
                  {/* Workspace Permissions */}
                  {/* TODO: Tie In Search */}
                  <PermissionContentList
                    selected={selectedPermissions}
                    key="workspace"
                    type="workspace"
                    onPermissionChange={(permissions) =>
                      handleWorkspacePermissionChange(permissions)
                    }
                  />
                  {/* From APIs */}
                  <p className="text-sm text-gray-10 ml-6 py-auto mt-1.5">From APIs</p>
                  {apis.map((api) => (
                    <PermissionContentList
                      selected={selectedPermissions}
                      key={api.id}
                      type="api"
                      api={api}
                      onPermissionChange={(permissions) =>
                        handleApiPermissionChange(api.id, permissions)
                      }
                    />
                  ))}
                </div>
              </ScrollArea>
            </div>
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
        </SheetDescription>
      </SheetContent>
    </Sheet>
  );
};
