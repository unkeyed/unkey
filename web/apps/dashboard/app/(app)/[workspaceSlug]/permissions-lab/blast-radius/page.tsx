"use client";

/**
 * Concept 2: Blast Radius Preview. A live panel answering "what does this
 * pattern actually cover" while you type, plus a diff view for pattern
 * changes. This is the trust-building surface for wildcards.
 */

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";
import { UrnText } from "../components/urn-display";
import { perm } from "../lib/mock-data";
import { CompareTab } from "./compare-tab";
import { ExploreTab } from "./explore-tab";

const CHEATSHEET_ROWS: { token: string; description: string; example: string }[] = [
  {
    token: "exact",
    description: "A path without wildcards matches exactly one resource, nothing else.",
    example: perm("keyspaces/ks_payments_prod", "create_key"),
  },
  {
    token: "*",
    description: "Matches exactly one path segment. It never reaches deeper levels.",
    example: perm("keyspaces/*/keys/*", "read_key"),
  },
  {
    token: "**",
    description: "Trailing only. Matches the base path itself and every descendant below it.",
    example: perm("projects/proj_storefront/**", "read_deployment"),
  },
  {
    token: "#*",
    description:
      'The action wildcard "*" is only valid on the global resource "**". The admin escape hatch.',
    example: perm("**", "*"),
  },
];

function GrammarCheatsheet() {
  return (
    <div className="rounded-lg border border-grayA-4 bg-grayA-2 p-5 flex flex-col gap-4">
      <span className="text-xs font-medium uppercase tracking-wide text-gray-10">
        Grammar cheatsheet
      </span>
      <div className="flex flex-col gap-4">
        {CHEATSHEET_ROWS.map((row) => (
          <div key={row.token} className="flex flex-col gap-1.5">
            <div className="flex items-baseline gap-2">
              <code className="rounded bg-grayA-3 px-1.5 py-0.5 font-mono text-xs font-semibold text-gray-12">
                {row.token}
              </code>
              <span className="text-sm text-gray-11">{row.description}</span>
            </div>
            <div className="overflow-x-auto">
              <UrnText value={row.example} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default function BlastRadiusPage() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Blast radius preview</PageHeaderTitle>
          <PageHeaderDescription>
            See exactly what a resource pattern covers before you grant it. Explore a single pattern
            live, or compare two patterns to preview a change.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col gap-10">
          <Tabs defaultValue="explore">
            <TabsList>
              <TabsTrigger value="explore">Explore</TabsTrigger>
              <TabsTrigger value="compare">Compare</TabsTrigger>
            </TabsList>
            <TabsContent value="explore" className="mt-6">
              <ExploreTab />
            </TabsContent>
            <TabsContent value="compare" className="mt-6">
              <CompareTab />
            </TabsContent>
          </Tabs>
          <GrammarCheatsheet />
        </div>
      </PageBody>
    </PageContainer>
  );
}
