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
import { RolesListControlCloud } from "./components/control-cloud";
import { RoleListControls } from "./components/controls";
import { RolesList } from "./components/table/roles-list";

const UpsertRoleDialog = dynamic(
  () => import("./components/upsert-role").then((mod) => mod.UpsertRoleDialog),
  { ssr: false },
);

export default function RolesPage() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Roles</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="primary" size="sm" className="px-3" onClick={() => setIsOpen(true)}>
            <Plus />
            Create new role
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col">
          <RoleListControls />
          <RolesListControlCloud />
          <RolesList />
        </div>
      </PageBody>
      <UpsertRoleDialog isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </PageContainer>
  );
}
