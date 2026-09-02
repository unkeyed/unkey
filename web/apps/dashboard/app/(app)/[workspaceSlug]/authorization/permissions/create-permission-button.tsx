"use client";
import { Button } from "@unkey/ui";
import dynamic from "next/dynamic";
import { IconPlusOutline18 } from "nucleo-ui-outline-18";
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
        <IconPlusOutline18 />
        New permission
      </Button>
      <UpsertPermissionDialog isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </div>
  );
}
