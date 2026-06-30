"use client";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useState } from "react";

const UpsertPermissionDialog = dynamic(
  () => import("./components/upsert-permission").then((mod) => mod.UpsertPermissionDialog),
  { ssr: false },
);

export function CreatePermissionButton() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative">
      <Button variant="primary" size="md" onClick={() => setIsOpen(true)}>
        <Plus iconSize="sm-regular" />
        New permission
      </Button>
      <UpsertPermissionDialog isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </div>
  );
}
