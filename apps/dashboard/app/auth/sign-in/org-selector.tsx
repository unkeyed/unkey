"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChevronsUpDown } from "lucide-react";
import type React from "react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { signIntoWorkspace } from "@/lib/auth/actions";
import { User, Organization } from "@/lib/auth/types";
  

interface OrgSelectorProps {
    user: User;
    organizations: Organization[];
}
  
export const OrgSelector: React.FC<OrgSelectorProps> = ({ user, organizations }) => {
    const getInitials = (user: User): string => {
        if (user.firstName && user.lastName) {
            return `${user.firstName[0]}${user.lastName[0]}`.toUpperCase();
          }
          
          // Fallback to first two characters of email
          return user.email.slice(0, 2).toUpperCase();
    };
      
    const displayName = user.fullName || user.email;

    return (
        <DropdownMenu>
          <DropdownMenuTrigger 
            data-state="open" 
            className="flex items-center justify-between w-full h-10 gap-2 px-2 overflow-hidden rounded-[0.625rem] bg-background border-border border hover:bg-background-subtle hover:cursor-pointer whitespace-nowrap ring-0 focus:ring-0 focus:outline-none text-content"
          >
            <div className="flex items-center gap-2 overflow-hidden whitespace-nowrap">
              <Avatar className="w-5 h-5">
                {user.avatarUrl ? (
                  <AvatarImage
                    src={user.avatarUrl}
                    alt={displayName}
                  />
                ) : null}
                <AvatarFallback className="flex items-center justify-center w-8 h-8 text-gray-700 bg-gray-100 border border-gray-500 rounded">
                  {getInitials(user)}
                </AvatarFallback>
              </Avatar>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="overflow-hidden text-sm font-medium text-ellipsis">
                    {displayName}
                  </span>
                </TooltipTrigger>
                <TooltipContent>
                  <span className="text-sm font-medium">{displayName}</span>
                </TooltipContent>
              </Tooltip>
            </div>
    
            <ChevronsUpDown className="hidden w-5 h-5 shrink-0 md:block [stroke-width:1px]" />
          </DropdownMenuTrigger>
          <DropdownMenuContent className="absolute left-0 w-96 max-sm:left-0">
            <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
            <DropdownMenuGroup>
              <ScrollArea className="h-96">
                {organizations.map((organization) => (
                  <DropdownMenuItem
                    key={organization.id}
                    className="flex items-center justify-between"
                    onClick={async () => {
                        await signIntoWorkspace(organization.id)
                    }}
                  >
                    <span>
                      {" "}{organization.name}
                    </span>
                  </DropdownMenuItem>
                ))}
              </ScrollArea>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    };