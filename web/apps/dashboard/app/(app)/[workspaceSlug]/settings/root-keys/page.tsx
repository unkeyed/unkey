"use client";
import { useFlag } from "@/lib/flags/provider";
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
import { useState } from "react";
import { BuilderAside } from "./components/builder/builder-aside";
import { RootKeysListControls } from "./components/controls";
import { CreateRootKeyButton } from "./components/dialog/create-rootkey-button";
import { RootKeysListBuilder } from "./components/table/root-keys-list-builder";
import { RootKeysListLegacy } from "./components/table/root-keys-list-legacy";

export default function RootKeysPage() {
  const rootKeyBuilder = useFlag("rootKeyBuilder");
  const [asideOpen, setAsideOpen] = useState(false);

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Root Keys</PageHeaderTitle>
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
              onClick={() => setAsideOpen(true)}
            >
              <Plus />
              New Root Key
            </Button>
          ) : (
            <CreateRootKeyButton />
          )}
        </PageHeaderActions>
      </PageHeader>
      {/* The table sizes itself to the viewport, so the body's default bottom
          padding would push the page past the app shell and add a second
          scrollbar. */}
      <PageBody className="pt-3 gap-3 pb-0">
        <RootKeysListControls />
        {rootKeyBuilder ? <RootKeysListBuilder /> : <RootKeysListLegacy />}
      </PageBody>
      {rootKeyBuilder ? (
        <BuilderAside isOpen={asideOpen} onClose={() => setAsideOpen(false)} />
      ) : null}
    </PageContainer>
  );
}
