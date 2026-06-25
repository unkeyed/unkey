"use client";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { RolesListControlCloud } from "./components/control-cloud";
import { RoleListControls } from "./components/controls";
import { RolesList } from "./components/table/roles-list";
import { CreateRoleButton } from "./create-role-button";

export default function RolesPage() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Roles</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateRoleButton />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col">
          <RoleListControls />
          <RolesListControlCloud />
          <RolesList />
        </div>
      </PageBody>
    </PageContainer>
  );
}
