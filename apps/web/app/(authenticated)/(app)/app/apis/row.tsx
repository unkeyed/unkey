"use client";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuGroup,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "@/components/ui/dropdown-menu";
import { Book, Key, MoreVertical, Rocket, Trash } from "lucide-react";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { Dialog } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";

type Props = {
  api: {
    id: string;
    name: string;
  };
};

export const Row: React.FC<Props> = ({ api }) => {
  return (
    <li
      key={api.id}
      className="flex items-center justify-between px-4 py-5 gap-x-6 md:px-6 lg:px-8"
    >
      <div>
        <p className="text-zinc-900 whitespace-nowrap">{api.name}</p>
        <div className="flex items-center mt-1 text-xs gap-x-2 leading-5 text-zinc-500">
          {/* <p className="whitespace-nowrap">
            Created at{" "}
            <time dateTime={apiKey.createdAt.toISOString()}>{apiKey.createdAt.toUTCString()}</time>
          </p> */}
        </div>
      </div>

      <Dialog>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              <MoreVertical />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-full lg:w-56" align="end" forceMount>
            <DropdownMenuGroup>
              {/* <DropdownMenuItem>
                                <User className="w-4 h-4 mr-2" />
                                <span>Profile</span>
                                <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
                            </DropdownMenuItem>
                            <DropdownMenuItem>
                                <CreditCard className="w-4 h-4 mr-2" />
                                <span>Billing</span>
                                <DropdownMenuShortcut>⌘B</DropdownMenuShortcut>
                            </DropdownMenuItem>
                            <DropdownMenuItem>
                                <Settings className="w-4 h-4 mr-2" />
                                <span>Settings</span>
                                <DropdownMenuShortcut>⌘S</DropdownMenuShortcut>
                            </DropdownMenuItem> */}

              <Link href={`/app/${api.id}`}>
                <DropdownMenuItem>
                  <Key className="w-4 h-4 mr-2" />
                  <span>Details</span>
                </DropdownMenuItem>
              </Link>

              {/* <DeleteKeyButton keyId={apiKey.id}>
                <DropdownMenuItem
                  onSelect={(e) => {
                    // This magically allows multiple dialogs in a dropdown menu, no idea why
                    e.preventDefault();
                  }}
                >
                  <Trash className="w-4 h-4 mr-2" />
                  <span>Revoke</span>
                </DropdownMenuItem>
              </DeleteKeyButton> */}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </Dialog>
    </li>
  );
};
