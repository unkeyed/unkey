import type { Parser } from "nuqs";
import { getTimestampFromRelative } from "../../lib/utils";
import type { FilterOperator, FilterUrlValue } from "../filter.types";

export const parseAsRelativeTime: Parser<string | null> = {
  parse: (str: string | null) => {
    if (!str) {
      return null;
    }

    try {
      // If that function doesn't throw it means we are safe
      getTimestampFromRelative(str);
      return str;
    } catch {
      return null;
    }
  },
  serialize: (value: string | null) => {
    if (!value) {
      return "";
    }
    return value;
  },
};

export const VALID_OPERATORS: FilterOperator[] = ["is", "contains"];

export const parseAsFilterValueArray = <TOperator extends FilterOperator>(
  operators: TOperator[] = VALID_OPERATORS as TOperator[],
): Parser<FilterUrlValue<TOperator>[]> => ({
  parse: (str: string | null) => {
    if (!str) {
      return [];
    }
    try {
      return str.split(",").map((item) => {
        const [operator, val] = item.split(/:(.+)/);
        if (!operators.includes(operator as TOperator)) {
          throw new Error("Invalid operator");
        }
        return {
          operator: operator as TOperator,
          value: val,
        };
      });
    } catch {
      return [];
    }
  },
  serialize: (value: FilterUrlValue<TOperator>[]) => {
    if (!value?.length) {
      return "";
    }
    return value.map((v) => `${v.operator}:${v.value}`).join(",");
  },
});

const VALID_DIRECTIONS = ["asc", "desc"] as const;
export type SortDirection = (typeof VALID_DIRECTIONS)[number];

export type SortUrlValue<TColumn extends string> = {
  column: TColumn;
  direction: SortDirection;
};

export const parseAsSortArray = <TColumn extends string>(): Parser<SortUrlValue<TColumn>[]> => ({
  parse: (str: string | null) => {
    if (!str) {
      return [];
    }
    try {
      return str.split(",").map((item) => {
        const [column, direction] = item.split(":");
        if (!column || !direction) {
          throw new Error("Invalid sort format");
        }
        if (!VALID_DIRECTIONS.includes(direction as SortDirection)) {
          throw new Error("Invalid direction");
        }
        return {
          column: column as TColumn,
          direction: direction as SortDirection,
        };
      });
    } catch {
      return [];
    }
  },
  serialize: (value: SortUrlValue<TColumn>[]) => {
    if (!value?.length) {
      return "";
    }
    return value.map((v) => `${v.column}:${v.direction}`).join(",");
  },
});
