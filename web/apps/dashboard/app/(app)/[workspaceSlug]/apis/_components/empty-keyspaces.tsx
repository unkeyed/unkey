"use client";

import { BookBookmark, Fingerprint, Gauge, Key, Nodes, ShieldKey } from "@unkey/icons";
import { Button, EmptyHero } from "@unkey/ui";
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
        <Gauge iconSize="md-medium" />
        <Fingerprint iconSize="md-medium" />
        <Key iconSize="md-thin" />
        <ShieldKey iconSize="md-medium" />
        <Nodes iconSize="md-medium" />
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
            <BookBookmark />
            Read the docs
          </Button>
        </a>
      </EmptyHero.Actions>
    </EmptyHero>
  );
}
