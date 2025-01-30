import type { z } from "zod";
import type { METHODS, STATUSES } from "./constants";
import type { filterFieldEnum, filterOperatorEnum } from "./filters.schema";

export type FilterOperator = z.infer<typeof filterOperatorEnum>;
export type FilterField = z.infer<typeof filterFieldEnum>;

export type HttpMethod = (typeof METHODS)[number];
export type ResponseStatus = (typeof STATUSES)[number];

export type FieldConfig = StringConfig | NumberConfig | StatusConfig;

export interface BaseFieldConfig<T extends string | number> {
  type: T extends string ? "string" : "number";
  operators: FilterOperator[];
}

export interface NumberConfig extends BaseFieldConfig<number> {
  type: "number";
  validate?: (value: number) => boolean;
  getColorClass?: (value: number) => string;
}

export interface StringConfig extends BaseFieldConfig<string> {
  type: "string";
  validValues?: readonly string[];
  validate?: (value: string) => boolean;
  getColorClass?: (value: string) => string;
}

export interface StatusConfig extends NumberConfig {
  type: "number";
  operators: ["is"];
  validate: (value: number) => boolean;
}

export type FilterFieldConfigs = {
  status: StatusConfig;
  methods: StringConfig;
  paths: StringConfig;
  host: StringConfig;
  requestId: StringConfig;
  startTime: NumberConfig;
  endTime: NumberConfig;
  since: StringConfig;
};

export type AllowedOperators<F extends FilterField> = FilterFieldConfigs[F]["operators"][number];

export type QuerySearchParams = {
  methods: FilterUrlValue[] | null;
  paths: FilterUrlValue[] | null;
  status: FilterUrlValue[] | null;
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  host: FilterUrlValue[] | null;
  requestId: FilterUrlValue[] | null;
};

export interface FilterUrlValue {
  value: string | number;
  operator: FilterOperator;
}

export interface FilterValue {
  id: string;
  field: FilterField;
  operator: FilterOperator;
  value: string | number | ResponseStatus | HttpMethod;
  metadata?: {
    colorClass?: string;
    icon?: React.ReactNode;
  };
}
