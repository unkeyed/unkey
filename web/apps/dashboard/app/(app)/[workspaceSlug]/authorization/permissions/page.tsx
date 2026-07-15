"use client";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { PermissionsListControlCloud } from "./components/control-cloud";
import { PermissionListControls } from "./components/controls";
import { PermissionsList } from "./components/table/permissions-list";
import { CreatePermissionButton } from "./create-permission-button";

export default function PermissionsPage() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Permissions</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreatePermissionButton />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col">
          <PermissionListControls />
          <PermissionsListControlCloud />
          <PermissionsList />
        </div>
      </PageBody>
    </PageContainer>
  );
}
