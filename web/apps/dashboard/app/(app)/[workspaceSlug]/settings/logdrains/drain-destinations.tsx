"use client";

import { ItemMedia } from "@unkey/ui";
import { IconEarthOutline18 } from "nucleo-ui-outline-18";
import type { ReactNode } from "react";
import { AxiomLogo } from "./axiom-logo";
import type { DrainKind } from "./drain-schema";

export const DESTINATIONS: ReadonlyArray<{
  kind: DrainKind;
  title: string;
  description: string;
  icon: ReactNode;
}> = [
  {
    kind: "http",
    title: "HTTP",
    description: "POST batches to an HTTPS endpoint",
    icon: <IconEarthOutline18 className="size-[18px]" />,
  },
  {
    kind: "axiom",
    title: "Axiom",
    description: "Ingest into an Axiom dataset",
    icon: <AxiomLogo className="size-[18px]" />,
  },
];

/** The destination icon, shared by the list and the create panel. */
export function DrainMedia({ kind }: { kind: DrainKind }) {
  return (
    <ItemMedia className="size-8 rounded-[10px] text-gray-12 ring-1 ring-grayA-4">
      {DESTINATIONS.find((destination) => destination.kind === kind)?.icon}
    </ItemMedia>
  );
}
