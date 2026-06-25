"use client";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useState } from "react";

const UpsertRoleDialog = dynamic(
  () => import("./components/upsert-role").then((mod) => mod.UpsertRoleDialog),
  { ssr: false },
);

export function CreateRoleButton() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative">
      <Button variant="primary" size="md" onClick={() => setIsOpen(true)}>
        <Plus iconSize="sm-regular" />
        New role
      </Button>
      <UpsertRoleDialog isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </div>
  );
}
