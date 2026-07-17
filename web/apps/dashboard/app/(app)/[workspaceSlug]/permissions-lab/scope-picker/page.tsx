"use client";

import {
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { useMemo, useState } from "react";
import { WORKSPACE_NAME } from "../lib/mock-data";
import { buildResourceTree, findNode } from "./model";
import { ResourceTree } from "./resource-tree";
import { ScopeCard } from "./scope-card";

export default function ScopePickerPage() {
  const tree = useMemo(() => buildResourceTree(), []);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const selectedNode = selectedPath ? findNode(tree, selectedPath) : null;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Scope Picker Tree</PageHeaderTitle>
          <PageHeaderDescription>
            Pick a resource, choose how far the grant reaches, then check the actions. Every choice
            shows the exact URN it produces, so the wildcard grammar is learned by watching, never
            by typing.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        {/* One-shot background pulse for the URN segment a scope change rewrote. */}
        <style>
          {
            "@keyframes permlab-scope-pulse { 0% { background-color: hsla(var(--warningA-6)); } 100% { background-color: transparent; } }"
          }
        </style>
        <div className="flex flex-col items-start gap-6 lg:flex-row">
          <aside className="w-full shrink-0 overflow-hidden rounded-lg border border-grayA-4 lg:sticky lg:top-4 lg:w-96">
            <div className="flex items-center justify-between border-b border-grayA-4 bg-grayA-2 px-3 py-2">
              <span className="text-xs font-medium uppercase tracking-wide text-gray-10">
                Resources
              </span>
              <span className="text-xs text-gray-9">{WORKSPACE_NAME}</span>
            </div>
            <div className="max-h-[72vh] overflow-y-auto p-2">
              <ResourceTree nodes={tree} selectedPath={selectedPath} onSelect={setSelectedPath} />
            </div>
          </aside>
          <div className="w-full min-w-0 flex-1">
            {selectedNode?.resource ? (
              <ScopeCard node={selectedNode} />
            ) : (
              <div className="rounded-lg border border-dashed border-grayA-4">
                <Empty>
                  <Empty.Title>Pick a resource to scope</Empty.Title>
                  <Empty.Description>
                    Select a keyspace, key, identity, or any other resource on the left. You get
                    three plain-language scope choices and the URN wildcards are generated for you.
                  </Empty.Description>
                </Empty>
              </div>
            )}
          </div>
        </div>
      </PageBody>
    </PageContainer>
  );
}
