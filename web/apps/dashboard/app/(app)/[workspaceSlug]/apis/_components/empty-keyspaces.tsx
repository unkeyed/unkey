"use client";

import { Button, EmptyHero } from "@unkey/ui";
import {
  IconBookBookmarkOutline18,
  IconFingerprintOutline18,
  IconGaugeOutline18,
  IconKeyOutline18,
  IconNodesOutline18,
  IconShieldKeyOutline18,
} from "nucleo-ui-outline-18";
import { CreateApiButton } from "./create-api-button";

export function EmptyKeyspaces({
  workspaceSlug,
  isNewApi,
}: {
  workspaceSlug: string;
  isNewApi: boolean;
}) {
  return (
    <EmptyHero>
      <EmptyHero.Icons>
        <IconGaugeOutline18 className="size-4" />
        <IconFingerprintOutline18 className="size-4" />
        <IconKeyOutline18 className="size-4" />
        <IconShieldKeyOutline18 className="size-4" />
        <IconNodesOutline18 className="size-4" />
      </EmptyHero.Icons>
      <EmptyHero.Title>Create your first keyspace</EmptyHero.Title>
      <EmptyHero.Description>
        You haven't created any keyspaces yet. Create one to get started.
      </EmptyHero.Description>
      <EmptyHero.Actions>
        <CreateApiButton defaultOpen={isNewApi} workspaceSlug={workspaceSlug} />
        <a
          href="https://www.unkey.com/docs/platform/apis/overview"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Button variant="outline" size="md">
            <IconBookBookmarkOutline18 />
            Read the docs
          </Button>
        </a>
      </EmptyHero.Actions>
    </EmptyHero>
  );
}
