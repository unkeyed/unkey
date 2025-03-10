export type ChartLabels = {
  title: string;
  primaryLabel: string;
  primaryKey: string;
  secondaryLabel: string;
  secondaryKey: string;
};

export type Selection = {
  start: string | number;
  end: string | number;
  startTimestamp?: number;
  endTimestamp?: number;
};

export type TimeseriesData = {
  originalTimestamp: number;
  [key: string]: any;
};
