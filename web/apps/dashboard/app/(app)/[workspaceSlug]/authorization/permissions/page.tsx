"use client";
import { Plus } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import { useState } from "react";
import { PermissionsListControlCloud } from "./components/control-cloud";
import { PermissionListControls } from "./components/controls";
import { PermissionsList } from "./components/table/permissions-list";

const UpsertPermissionDialog = dynamic(
  () => import("./components/upsert-permission").then((mod) => mod.UpsertPermissionDialog),
  { ssr: false },
);

export default function PermissionsPage() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Permissions</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="primary" size="sm" className="px-3" onClick={() => setIsOpen(true)}>
            <Plus />
            Create new permission
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col">
          <PermissionListControls />
          <PermissionsListControlCloud />
          <PermissionsList />
        </div>
      </PageBody>
      <UpsertPermissionDialog isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </PageContainer>
  );
}
