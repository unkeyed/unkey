"use client";
import { DeleteKeyButton } from "./DeleteKey";
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
  apiKey: {
    id: string;
    name?: string;
    start: string;
    createdAt: Date;
  };
};

export const Row: React.FC<Props> = ({ apiKey }) => {
  return (
    <li
      key={apiKey.id}
      className="flex items-center justify-between px-4 py-5 gap-x-6 md:px-6 lg:px-8"
    >
      <div>
        <p className="text-zinc-200 whitespace-nowrap">{apiKey.name}</p>
        <div className="flex items-center mt-1 text-xs leading-5 gap-x-2 text-zinc-500">
          <p className="whitespace-nowrap">
            Created at{" "}
            <time dateTime={apiKey.createdAt.toISOString()}>{apiKey.createdAt.toUTCString()}</time>
          </p>
        </div>
      </div>
      <div>
        <Badge variant="secondary">{apiKey.start}...</Badge>
      </div>
      <Dialog>
        <DropdownMenu>
          <DropdownMenuTrigger>
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

              <Link href={`/app/settings/keys/${apiKey.id}`}>
                <DropdownMenuItem>
                  <Key className="w-4 h-4 mr-2" />
                  <span>Details</span>
                </DropdownMenuItem>
              </Link>

              <DeleteKeyButton keyId={apiKey.id}>
                <DropdownMenuItem
                  onSelect={(e) => {
                    // This magically allows multiple dialogs in a dropdown menu, no idea why
                    e.preventDefault();
                  }}
                >
                  <Trash className="w-4 h-4 mr-2" />
                  <span>Revoke</span>
                </DropdownMenuItem>
              </DeleteKeyButton>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </Dialog>
    </li>
  );
};
