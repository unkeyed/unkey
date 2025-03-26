"use client";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { completeOrgSelection } from "../actions";
import type { Organization } from "@/lib/auth/types";
import { Button } from "@unkey/ui";
import type React from "react";
import { useState, useEffect } from "react";

interface OrgSelectorProps {
  organizations: Organization[];
}

export const OrgSelector: React.FC<OrgSelectorProps> = ({ organizations }) => {
  const [selected, setSelected] = useState<string>();
  const [isOpen, setIsOpen] = useState(false);
  const [clientReady, setClientReady] = useState(false);

  // Set client ready after hydration
  useEffect(() => {
    setClientReady(true);
    // Only open the dialog after hydration to prevent hydration mismatch
    setIsOpen(true);
  }, []);

  const handleContinue = async () => {
    if (!selected) {
      return;
    }
    await completeOrgSelection(selected);
    setIsOpen(false);
  };

  return (
    <Dialog
      open={clientReady && isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
      }}
    >
      <DialogContent className="border-border w-11/12">
        <DialogHeader>
          <DialogTitle>Workspace Selection</DialogTitle>
          <DialogDescription>Select a workspace to continue authentication:</DialogDescription>
        </DialogHeader>
        <Select onValueChange={(orgId) => setSelected(orgId)} value={selected}>
          <SelectTrigger>
            <SelectValue placeholder="Select a Workspace" />
          </SelectTrigger>
          <SelectContent>
            {organizations.map((org) => (
              <SelectItem key={org.id} value={org.id}>
                {org.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="primary" onClick={handleContinue} disabled={!selected}>
          Continue with Sign-In
        </Button>
      </DialogContent>
    </Dialog>
  );
};