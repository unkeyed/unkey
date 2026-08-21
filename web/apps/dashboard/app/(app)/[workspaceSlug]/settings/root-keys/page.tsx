"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
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
import Link from "next/link";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { ROOT_KEY_MESSAGES } from "./components/dialog/constants";
import { CreateRootKeyButton } from "./components/dialog/create-rootkey-button";
import { RootKeysList } from "./components/table/root-keys-list";

export default function RootKeysPage() {
  const workspace = useWorkspaceNavigation();
  const rootKeyBuilder = useFlag("rootKeyBuilder");

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Root Keys</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          {rootKeyBuilder ? (
            <Button
              variant="primary"
              size="sm"
              className="px-3 rounded-md"
              render={<Link href={routes.settings.rootKeyNew({ workspaceSlug: workspace.slug })} />}
            >
              <Plus />
              {ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY}
            </Button>
          ) : (
            <CreateRootKeyButton />
          )}
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col">
          <RootKeysListControls />
          <RootKeysListControlCloud />
          <RootKeysList />
        </div>
      </PageBody>
    </PageContainer>
  );
}
