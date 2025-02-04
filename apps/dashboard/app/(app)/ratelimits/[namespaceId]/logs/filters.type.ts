import type { z } from "zod";
import type { filterFieldEnum, filterOperatorEnum } from "./filters.schema";

export type FilterOperator = z.infer<typeof filterOperatorEnum>;
export type FilterField = z.infer<typeof filterFieldEnum>;

export type FieldConfig = StringConfig | NumberConfig;

export interface BaseFieldConfig<T extends string | number> {
  type: T extends string ? "string" : "number";
  operators: FilterOperator[];
}

export interface NumberConfig extends BaseFieldConfig<number> {
  type: "number";
  validate?: (value: number) => boolean;
}

export interface StringConfig extends BaseFieldConfig<string> {
  type: "string";
  validValues?: readonly string[];
  validate?: (value: string) => boolean;
}

export type FilterFieldConfigs = {
  startTime: NumberConfig;
  endTime: NumberConfig;
  since: StringConfig;
  identifiers: StringConfig;
  requestIds: StringConfig;
  status: StringConfig;
};

export type QuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  identifiers: FilterUrlValue[] | null;
  requestIds: FilterUrlValue[] | null;
  status: FilterUrlValue[] | null;
};

export type FilterUrlValue = Pick<FilterValue, "value" | "operator">;

export type FilterValue = {
  id: string;
  field: FilterField;
  operator: FilterOperator;
  value: string | number;
};
