import { format } from "date-fns";

export const defaultFormatValue = (value: string | number, field: string): string => {
  if (typeof value === "number" && (field === "startTime" || field === "endTime")) {
    return format(value, "MMM d, yyyy HH:mm:ss");
  }
  return String(value);
};

export const formatOperator = (operator: string, field: string): string => {
  if (field === "since" && operator === "is") {
    return "Last";
  }
  return operator;
};
