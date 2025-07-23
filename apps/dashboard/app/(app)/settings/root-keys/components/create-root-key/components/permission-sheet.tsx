"use client";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTrigger,
} from "@/components/ui/sheet";

import { ScrollArea } from "@/components/ui/scroll-area";
import { useRef, useState } from "react";
import { PermissionContentList } from "./permission-list";
import { SearchPermissions } from "./search-permissions";

type PermissionSheetProps = {
  children: React.ReactNode;
  apis:
    | {
        id: string;
        name: string;
      }[]
    | [];
};
export const PermissionSheet = ({ children, apis }: PermissionSheetProps) => {
  const [open, setOpen] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [search, setSearch] = useState<string | undefined>(undefined);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setIsProcessing(true);
    setSearch(e.target.value);
    setTimeout(() => {
      setIsProcessing(false);
    }, 1000);
  };

  const handleOpenChange = (open: boolean) => {
    setOpen((prev) => !prev);
  };
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
          <ScrollArea className="flex flex-col h-full">
            <div className="flex flex-col h-full pt-0 mt-0 gap-1">
              {/* Workspace Permissions */}
              {/* TODO: Tie In Search */}
              {/* TODO: Return permissions to the form */}
              <PermissionContentList type="workspace" />
              {/* From APIs */}
              {/* TODO: add real API list */}
              <p className="text-sm text-gray-10 ml-6 py-auto mt-1.5">From APIs</p>
              {fakeApis.map((api) => (
                <PermissionContentList type="api" api={api} />
              ))}
            </div>
          </ScrollArea>
        </SheetDescription>
      </SheetContent>
    </Sheet>
  );
};

// TODO: Remove this when we have a real API list
const fakeApis = [
  {
    id: "api_12345",
    name: "API 12345",
  },
  {
    id: "api_12346",
    name: "API 12346",
  },
  {
    id: "api_12347",
    name: "API 12347",
  },
];
