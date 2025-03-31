import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { OUTCOME_BACKGROUND_COLORS } from "../../../constants";
import { formatOutcomeName } from "../../../utils";

/**
 * Maps CSS variables from our design system to actual color values
 * @param cssVar CSS variable name (e.g., "success-9")
 * @returns HSL string compatible with chart libraries
 */
export function cssVarToChartColor(cssVar: string): string {
  // Remove "bg-" prefix if present
  const cleanVar = cssVar.replace("bg-", "");
  return `hsl(var(--${cleanVar}))`;
}

/**
 * Create chart configuration for all outcomes or a subset
 * @param includedOutcomes Specific outcomes to include (defaults to all)
 * @returns Configuration object for charts
 */
export function createOutcomeChartConfig(includedOutcomes?: string[]) {
  const config: Record<string, { label: string; color: string }> = {
    success: {
      label: formatOutcomeName("VALID"),
      color: cssVarToChartColor("accent-4"),
    },
  };

  // Default to all non-valid outcomes if none specified
  const outcomesToInclude =
    includedOutcomes ||
    KEY_VERIFICATION_OUTCOMES.filter((outcome) => outcome !== "VALID" && outcome !== "");

  // Add each outcome as a chart series option
  outcomesToInclude.forEach((outcome) => {
    if (outcome === "VALID" || outcome === "") {
      return; // Skip VALID (already added) and empty string
    }

    // Convert to the format used in our timeseries data (snake_case)
    const key = outcome.toLowerCase();
    const colorClass = OUTCOME_BACKGROUND_COLORS[outcome] || "bg-accent-4";

    config[key] = {
      label: formatOutcomeName(outcome),
      color: cssVarToChartColor(colorClass),
    };
  });

  return config;
}

/**
 * Get outcome color for chart visualization
 * @param outcome Outcome string
 * @returns CSS color compatible with chart libraries
 */
export function getOutcomeChartColor(outcome: string): string {
  const colorClass = OUTCOME_BACKGROUND_COLORS[outcome] || "bg-accent-9";
  return cssVarToChartColor(colorClass);
}
