import type { FieldConfig, NumberConfig, StringConfig } from "../filter.types";

// Type guards
export function isNumberConfig(config: FieldConfig): config is NumberConfig {
  return config.type === "number";
}
export function isStringConfig(config: FieldConfig): config is StringConfig {
  return config.type === "string";
}
