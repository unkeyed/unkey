import type { OptionsType } from "./types";

export const DEFAULT_OPTIONS: OptionsType = [
  {
    id: 1,
    value: "1m",
    display: "Last minute",
    checked: false,
  },
  {
    id: 2,
    value: "5m",
    display: "Last 5 minutes",
    checked: false,
  },
  {
    id: 3,
    value: "15m",
    display: "Last 15 minutes",
    checked: false,
  },
  {
    id: 4,
    value: "30m",
    display: "Last 30 minutes",
    checked: false,
  },
  {
    id: 5,
    value: "1h",
    display: "Last 1 hour",
    checked: false,
  },
  {
    id: 6,
    value: "3h",
    display: "Last 3 hours",
    checked: false,
  },
  {
    id: 7,
    value: "6h",
    display: "Last 6 hours",
    checked: false,
  },
  {
    id: 8,
    value: "12h",
    display: "Last 12 hours",
    checked: false,
  },
  {
    id: 9,
    value: "24h",
    display: "Last 24 hours",
    checked: false,
  },
  {
    id: 10,
    value: "2d",
    display: "Last 2 days",
    checked: false,
  },
  {
    id: 11,
    value: "3d",
    display: "Last 3 days",
    checked: false,
  },
  {
    id: 12,
    value: "1w",
    display: "Last week",
    checked: false,
  },
  {
    id: 13,
    value: "2w",
    display: "Last 2 weeks",
    checked: false,
  },
  {
    id: 14,
    value: "4w",
    display: "Last 4 weeks",
    checked: false,
  },
  {
    id: 15,
    value: undefined,
    display: "Custom",
    checked: false,
  },
];

export const CUSTOM_OPTION_ID = DEFAULT_OPTIONS.find((o) => o.value === undefined)?.id;
