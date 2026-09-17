import type { ComponentType } from "react";
import type { VariantProps } from "./types";
import { RailCards, RailCardsTwoLine, RailPlain } from "./variants/rail-cards";
import { RailActivity, RailDeploysOnly } from "./variants/rail-activity";
import { RailFlat, RailQuiet, RailSearch, RailSummary } from "./variants/rail-flat";
import { BandColumns, RailNarrow, RailTable } from "./variants/rail-table";

export type VariantId =
  | "cards"
  | "cards-two-line"
  | "plain"
  | "flat"
  | "summary"
  | "table"
  | "quiet"
  | "narrow"
  | "search"
  | "band"
  | "activity"
  | "deploys";

export type Variant = {
  id: VariantId;
  name: string;
  note: string;
  /** "band" drops out of the rail and renders under the projects grid. */
  placement: "rail" | "band";
  width?: string;
  Component: ComponentType<VariantProps>;
};

export const VARIANTS: Variant[] = [
  {
    id: "activity",
    name: "Activity first",
    note: "Vercel's column: recently shipped, recent previews, then resources.",
    placement: "rail",
    Component: RailActivity,
  },
  {
    id: "deploys",
    name: "Deploys only",
    note: "Just the two deploy lists, to judge the row on its own.",
    placement: "rail",
    Component: RailDeploysOnly,
  },
  {
    id: "cards",
    name: "Cards, tightened",
    note: "Today's shape with the padding taken out. One line per resource.",
    placement: "rail",
    Component: RailCards,
  },
  {
    id: "cards-two-line",
    name: "Cards, two line",
    note: "Keeps key counts and the project chip on a second line.",
    placement: "rail",
    Component: RailCardsTwoLine,
  },
  {
    id: "plain",
    name: "No chrome",
    note: "Vercel's left column: a label, then rows straight on the page.",
    placement: "rail",
    Component: RailPlain,
  },
  {
    id: "flat",
    name: "Most used",
    note: "One ranked list across both products. Glyph carries the type.",
    placement: "rail",
    Component: RailFlat,
  },
  {
    id: "summary",
    name: "Totals first",
    note: "A line per product; open one to see the names behind it.",
    placement: "rail",
    Component: RailSummary,
  },
  {
    id: "table",
    name: "Table",
    note: "Aligned columns with a header row. Scan the numbers.",
    placement: "rail",
    Component: RailTable,
  },
  {
    id: "quiet",
    name: "Quiet",
    note: "Hairlines, dimmed names, no borders. Recedes next to the grid.",
    placement: "rail",
    Component: RailQuiet,
  },
  {
    id: "narrow",
    name: "Narrow",
    note: "240px. Name and number only.",
    placement: "rail",
    width: "lg:w-[240px]",
    Component: RailNarrow,
  },
  {
    id: "search",
    name: "Search first",
    note: "Filter box on top. The answer for a 2,000-keyspace default project.",
    placement: "rail",
    Component: RailSearch,
  },
  {
    id: "band",
    name: "Band under grid",
    note: "Stops being a rail. Projects get the full page width.",
    placement: "band",
    Component: BandColumns,
  },
];

export const DEFAULT_VARIANT: VariantId = "flat";

export function findVariant(id: string | null): Variant {
  return VARIANTS.find((variant) => variant.id === id) ?? VARIANTS[0];
}
