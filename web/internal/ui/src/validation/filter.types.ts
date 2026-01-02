import type { ReactNode } from "react";
import { z } from "zod";

// Our default filterOperators that we use in query params e.g. "path=is:/oz/refactors"
export const filterOperatorEnum = z.enum(["is", "contains", "startsWith", "endsWith"]);

export type FilterOperator = z.infer<typeof filterOperatorEnum>;

export type BaseFieldConfig<T extends string | number, TOperator extends string = string> = {
  type: T extends string ? "string" : "number";
  operators: ReadonlyArray<TOperator>;
};

export type NumberConfig<TOperator extends string = string> = BaseFieldConfig<number, TOperator> & {
  type: "number";
  validate?: (value: number) => boolean;
  getColorClass?: (value: number) => string;
};

interface SortableConfig {
  sortable?: boolean;
  defaultSortDirection?: "asc" | "desc";
  sortWeight?: number;
}

export interface SortableNumberConfig<TOperator extends string = string>
  extends NumberConfig<TOperator>,
    SortableConfig {}
export interface SortableStringConfig<TOperator extends string = string>
  extends StringConfig<TOperator>,
    SortableConfig {}

export type StringConfig<TOperator extends string = string> = BaseFieldConfig<string, TOperator> & {
  type: "string";
  validValues?: readonly string[];
  validate?: (value: string) => boolean;
  getColorClass?: (value: string) => string;
};

export type FieldConfig<TOperator extends string = string> =
  | StringConfig<TOperator>
  | NumberConfig<TOperator>;

export type FilterUrlValue<TOperator extends FilterOperator = FilterOperator> = {
  value: string | number;
  operator: TOperator;
};

export type FilterValue<
  TField extends string = string,
  TOperator extends FilterOperator = FilterOperator,
  TValue extends string | number = string | number,
> = {
  id: string;
  field: TField;
  operator: TOperator;
  value: TValue;
  metadata?: {
    colorClass?: string;
    icon?: ReactNode;
  };
};
