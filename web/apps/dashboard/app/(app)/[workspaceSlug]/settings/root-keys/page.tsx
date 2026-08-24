"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { BookBookmark, Plus } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  buttonVariants,
} from "@unkey/ui";
import Link from "next/link";
import { RootKeysListControls } from "./components/controls";
import { ROOT_KEY_MESSAGES } from "./components/dialog/constants";
import { CreateRootKeyButton } from "./components/dialog/create-rootkey-button";
import { RootKeysList } from "./components/table/root-keys-list";

export default function RootKeysPage() {
  const workspace = useWorkspaceNavigation();
  const rootKeyBuilder = useFlag("rootKeyBuilder");

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Root keys</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <a
            href="https://www.unkey.com/docs/security/overview#root-keys"
            target="_blank"
            rel="noopener noreferrer"
            className={buttonVariants({ variant: "outline", size: "sm", className: "px-3" })}
          >
            <BookBookmark />
            Documentation
          </a>
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
      <PageBody className="pt-3 gap-3">
        <RootKeysListControls />
        <RootKeysList />
      </PageBody>
    </PageContainer>
  );
}
