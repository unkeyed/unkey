import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_GROUP_NAMES,
  type DeploymentStatusGroup,
  isDeploymentStatusGroup,
} from "@/lib/collections/deploy/deployment-status";
import { z } from "zod";

export const GROUPED_DEPLOYMENT_STATUSES = DEPLOYMENT_STATUS_GROUP_NAMES;
export type GroupedDeploymentStatus = DeploymentStatusGroup;

export const DEPLOYMENT_STATUS_META: Record<
  GroupedDeploymentStatus,
  { label: string; colorClass: string }
> = {
  ready: { label: "Ready", colorClass: DEPLOYMENT_GROUP_COLOR.ready },
  failed: { label: "Failed", colorClass: DEPLOYMENT_GROUP_COLOR.failed },
  building: { label: "Building", colorClass: DEPLOYMENT_GROUP_COLOR.building },
  queued: { label: "Queued", colorClass: DEPLOYMENT_GROUP_COLOR.queued },
  blocked: { label: "Awaiting Approval", colorClass: DEPLOYMENT_GROUP_COLOR.blocked },
  cancelled: { label: "Cancelled", colorClass: DEPLOYMENT_GROUP_COLOR.cancelled },
  superseded: { label: "Superseded", colorClass: DEPLOYMENT_GROUP_COLOR.superseded },
  stopped: { label: "Stopped", colorClass: DEPLOYMENT_GROUP_COLOR.stopped },
};

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
    getColorClass: (value) =>
      isDeploymentStatusGroup(value) ? DEPLOYMENT_STATUS_META[value].colorClass : "bg-info-9",
  },
  environment: {
    type: "string",
    operators: ["is"],
  },
  branch: {
    type: "string",
    operators: ["is"],
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
