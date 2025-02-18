import { getTimestampFromRelative } from "@/lib/utils";
import type { Parser } from "nuqs";
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
