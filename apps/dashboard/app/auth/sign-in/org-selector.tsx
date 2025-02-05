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
import { completeOrgSelection, signIntoWorkspace } from "@/lib/auth/actions";
import { User, Organization } from "@/lib/auth/types";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";
import { Button } from "@unkey/ui";
import { Dialog, DialogContent, DialogDescription, DialogHeader } from "@/components/ui/dialog";
  

interface OrgSelectorProps {
    organizations: Organization[];
}
  
export const OrgSelector: React.FC<OrgSelectorProps> = ({ organizations }) => {
    const [selected, setSelected] = useState<string>();

    return (
      <Dialog defaultOpen>
             <DialogContent className="border-border w-11/12 max-sm:">
             <DialogHeader>
              <DialogDescription>
                Select a workspace to continue authentication:
              </DialogDescription>
             </DialogHeader>
                <Select 
        onValueChange={(orgId) => setSelected(orgId)} 
        value={selected}>
      <SelectTrigger>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
      {organizations.map((org) => (
        <SelectItem key={org.id} value={org.id}>
          {org.name}
        </SelectItem>
      ))}
      </SelectContent>
    </Select>
        <Button variant="primary" onClick={() => {
          if (!selected) return null;
          return completeOrgSelection(selected)
        }} 
        disabled={!selected}>Continue with Sign-In</Button>
             </DialogContent>
            </Dialog>
      
    )};