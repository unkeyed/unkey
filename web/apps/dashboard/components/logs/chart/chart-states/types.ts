/**
 * Shared types for chart loading and error states
 *
 * This file consolidates types used across different chart implementations
 * to provide consistent error and loading experiences.
 */

/**
 * Metric configuration for displaying chart data
 */
export type ChartMetric = {
  key: string;
  label: string;
  color: string;
  formatter?: (value: number) => string;
};

/**
 * Labels and configuration for timeseries charts
 */
export type TimeseriesChartLabels = {
  title: string;
  rangeLabel: string;
  metrics: ChartMetric[];
  showRightSide?: boolean;
  reverse?: boolean;
};

/**
 * Variant types for chart state components
 * - "simple": Minimal error/loading display (used in stats cards)
 * - "compact": With time labels and fixed height (used in logs chart)
 * - "full": Complete layout with header, metrics, and footer (used in overview charts)
 */
export type ChartStateVariant = "simple" | "compact" | "full";

/**
 * Props for the ChartError component
 */
export type ChartErrorProps = {
  variant?: ChartStateVariant;
  message?: string;
  labels?: TimeseriesChartLabels;
  height?: number;
  className?: string;
};

/**
 * Props for the ChartLoading component
 */
export type ChartLoadingProps = {
  variant?: ChartStateVariant;
  labels?: TimeseriesChartLabels;
  height?: number;
  className?: string;
  animate?: boolean;
  dataPoints?: number;
};

/**
 * Props for the ChartEmpty component
 */
export type ChartEmptyProps = {
  variant?: ChartStateVariant;
  message?: string;
  labels?: TimeseriesChartLabels;
  height?: number;
  className?: string;
};
