import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

export const DEPLOYMENT_STATUSES = [
  "pending", "building", "deploying", "network", "ready", "failed"
] as const;

// Define grouped statuses for client filtering
const GROUPED_DEPLOYMENT_STATUSES = [
  "pending",
  "building", // represents all building states
  "ready",
  "failed",
] as const;

const DEPLOYMENT_ENVIRONMENTS = ["production", "preview"] as const;

export type DeploymentStatus = (typeof DEPLOYMENT_STATUSES)[number];
export type GroupedDeploymentStatus = (typeof GROUPED_DEPLOYMENT_STATUSES)[number];
export type DeploymentEnvironment = (typeof DEPLOYMENT_ENVIRONMENTS)[number];

const allOperators = ["is", "contains"] as const;

export const deploymentListFilterOperatorEnum = z.enum(allOperators);
export type DeploymentListFilterOperator = z.infer<typeof deploymentListFilterOperatorEnum>;

export type FilterFieldConfigs = {
  status: StringConfig<DeploymentListFilterOperator>;
  environment: StringConfig<DeploymentListFilterOperator>;
  branch: StringConfig<DeploymentListFilterOperator>;
  startTime: NumberConfig<DeploymentListFilterOperator>;
  endTime: NumberConfig<DeploymentListFilterOperator>;
  since: StringConfig<DeploymentListFilterOperator>;
};

export const deploymentListFilterFieldConfig: FilterFieldConfigs = {
  status: {
    type: "string",
    operators: ["is"],
    validValues: GROUPED_DEPLOYMENT_STATUSES,
    getColorClass: (value) => {
      if (value === "completed") {
        return "bg-success-9";
      }
      if (value === "failed") {
        return "bg-error-9";
      }
      if (value === "pending") {
        return "bg-gray-9";
      }
      return "bg-info-9"; // building
    },
  },
  environment: {
    type: "string",
    operators: ["is"],
    validValues: DEPLOYMENT_ENVIRONMENTS,
  },
  branch: {
    type: "string",
    operators: ["contains"],
  },
  startTime: {
    type: "number",
    operators: ["is"],
  },
  endTime: {
    type: "number",
    operators: ["is"],
  },
  since: {
    type: "string",
    operators: ["is"],
  },
};

// Mapping function to expand grouped statuses to actual statuses
export const expandGroupedStatus = (groupedStatus: GroupedDeploymentStatus): DeploymentStatus[] => {
  switch (groupedStatus) {
    case "pending":
      return ["pending"];
    case "building":
      return [
        "building",
        "deploying",
        "network",
      ];
    case "ready":
      return ["ready"];
    case "failed":
      return ["failed"];
    default:
      throw new Error(`Unknown grouped status: ${groupedStatus}`);
  }
};

const allFilterFieldNames = Object.keys(
  deploymentListFilterFieldConfig,
) as (keyof FilterFieldConfigs)[];

if (allFilterFieldNames.length === 0) {
  throw new Error("deploymentListFilterFieldConfig must contain at least one field definition.");
}

const [firstFieldName, ...restFieldNames] = allFilterFieldNames;

export const deploymentListFilterFieldEnum = z.enum([firstFieldName, ...restFieldNames]);

export const deploymentListFilterFieldNames = allFilterFieldNames;
export type DeploymentListFilterField = z.infer<typeof deploymentListFilterFieldEnum>;

export const deploymentListFilterOutputSchema = createFilterOutputSchema(
  deploymentListFilterFieldEnum,
  deploymentListFilterOperatorEnum,
  deploymentListFilterFieldConfig,
);

export type DeploymentListFilterUrlValue = {
  value: string;
  operator: DeploymentListFilterOperator;
};

export type DeploymentListFilterValue = FilterValue<
  DeploymentListFilterField,
  DeploymentListFilterOperator
>;

export type DeploymentListQuerySearchParams = {
  status: DeploymentListFilterUrlValue[] | null;
  environment: DeploymentListFilterUrlValue[] | null;
  branch: DeploymentListFilterUrlValue[] | null;
  startTime: number | null;
  endTime: number | null;
  since: string | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<DeploymentListFilterOperator>(
  [...allOperators],
);
