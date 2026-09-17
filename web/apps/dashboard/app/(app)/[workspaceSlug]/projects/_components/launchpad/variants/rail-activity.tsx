"use client";

import { IconFingerprintOutline18 } from "@unkey/icons";
import { RowLink, RowSkeleton, Section, SummaryRow, fmt } from "../parts";
import { DeployList } from "../recent-deploys";
import type { VariantProps } from "../types";
import { GetStarted, Value } from "./shared";

/**
 * Vercel's left column, read across: what shipped, what is in preview, then the
 * resources to jump into.
 */
export function RailActivity({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} count={6} />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return (
      <Section title="Get started" chrome={options.chrome}>
        <GetStarted density={options.density} />
      </Section>
    );
  }
  return (
    <>
      <DeployList
        title="Recently shipped"
        chrome={options.chrome}
        deploys={model.readyDeploys}
        empty="Nothing has reached production yet."
      />
      <DeployList
        title="Recent previews"
        chrome={options.chrome}
        deploys={model.previewDeploys}
        empty="No preview branches deployed."
      />
      <Section
        title="Most used"
        chrome={options.chrome}
        subtitle={`last ${model.windowHours}h`}
        count={model.rows.length}
      >
        {model.rows.slice(0, 5).map((row) => (
          <RowLink key={row.id} row={row} density={options.density}>
            <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
            <Value row={row} />
          </RowLink>
        ))}
        {model.identityCount > 0 && (
          <SummaryRow
            icon={<IconFingerprintOutline18 className="size-3 shrink-0 text-gray-9" />}
            label="Identities"
            value={fmt(model.identityCount)}
            href={model.identitiesHref}
            density={options.density}
          />
        )}
      </Section>
    </>
  );
}

/** Deploy activity only, for judging the row itself without the rail around it. */
export function RailDeploysOnly({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} count={6} />;
  }
  return (
    <>
      <DeployList
        title="Recently shipped"
        chrome={options.chrome}
        deploys={model.readyDeploys}
        limit={5}
        empty="Nothing has reached production yet."
      />
      <DeployList
        title="Recent previews"
        chrome={options.chrome}
        deploys={model.previewDeploys}
        limit={5}
        empty="No preview branches deployed."
      />
    </>
  );
}
