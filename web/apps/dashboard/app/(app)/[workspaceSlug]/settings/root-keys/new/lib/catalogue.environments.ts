import { INSTANCE_TOKEN, type ScopeCatalogue, actionGrant } from "./catalogue.types";

const ENVIRONMENT_PATH = INSTANCE_TOKEN;
const DEPLOYMENT_PATH = `${ENVIRONMENT_PATH}/deployments/*`;
const ENVIRONMENT_ALL_PATH = "projects/*/apps/*/environments/*";

export const environmentsCatalogue: ScopeCatalogue = {
  scope: "environments",
  label: "Specific environments",
  allLabel: "All environments",
  instanceNoun: "environments",
  groups: [
    {
      id: "environment",
      label: "Environments",
      rows: [
        {
          id: "environment",
          label: "Environments",
          path: ENVIRONMENT_PATH,
          allPath: ENVIRONMENT_ALL_PATH,
          resource: "environment",
          actions: {
            read_environment: actionGrant("read_environment", "environments:read"),
            write_environment: actionGrant("write_environment", "environments:write"),
            delete_environment: actionGrant("delete_environment", "environments:delete"),
          },
        },
        {
          id: "variable",
          label: "Environment variables",
          path: `${ENVIRONMENT_PATH}/variables/*`,
          allPath: `${ENVIRONMENT_ALL_PATH}/variables/*`,
          resource: "variable",
          actions: {
            read_environment_variable: actionGrant(
              "read_environment_variable",
              "environment_variables:read",
            ),
            write_environment_variable: actionGrant(
              "write_environment_variable",
              "environment_variables:write",
            ),
            delete_environment_variable: actionGrant(
              "delete_environment_variable",
              "environment_variables:delete",
            ),
          },
        },
        {
          id: "domain",
          label: "Domains",
          path: `${ENVIRONMENT_PATH}/domains/*`,
          allPath: `${ENVIRONMENT_ALL_PATH}/domains/*`,
          resource: "domain",
          actions: {
            read_domain: actionGrant("read_domain", "domains:read"),
            write_domain: actionGrant("write_domain", "domains:write"),
            delete_domain: actionGrant("delete_domain", "domains:delete"),
          },
        },
      ],
    },
    {
      id: "deployments",
      label: "Deployments",
      rows: [
        {
          id: "deployment",
          label: "Deployments",
          path: DEPLOYMENT_PATH,
          allPath: `${ENVIRONMENT_ALL_PATH}/deployments/*`,
          resource: "deployment",
          actions: {
            read_deployment: actionGrant("read_deployment", "deployments:read"),
            write_deployment: actionGrant("write_deployment", "deployments:write"),
            delete_deployment: actionGrant("delete_deployment", "deployments:delete"),
            start_deployment: actionGrant("start_deployment", "deployments:start"),
            stop_deployment: actionGrant("stop_deployment", "deployments:stop"),
          },
        },
        {
          id: "deployment_log",
          label: "Runtime logs",
          path: `${DEPLOYMENT_PATH}/logs`,
          allPath: `${ENVIRONMENT_ALL_PATH}/deployments/*/logs`,
          resource: "deployment_log",
          actions: {
            read_deployment_logs: actionGrant("read_deployment_logs", "deployment_logs:read"),
          },
        },
      ],
    },
    {
      id: "gateway",
      label: "Gateway",
      rows: [
        {
          id: "gateway_log",
          label: "HTTP request logs",
          path: `${ENVIRONMENT_PATH}/gateway/logs`,
          allPath: `${ENVIRONMENT_ALL_PATH}/gateway/logs`,
          resource: "gateway_log",
          actions: {
            read_gateway_logs: actionGrant("read_gateway_logs", "gateway_logs:read"),
          },
        },
        {
          id: "gateway_policy",
          label: "Gateway policies",
          path: `${ENVIRONMENT_PATH}/gateway/policies/*`,
          allPath: `${ENVIRONMENT_ALL_PATH}/gateway/policies/*`,
          resource: "gateway_policy",
          actions: {
            read_gateway_policy: actionGrant("read_gateway_policy", "gateway_policies:read"),
            write_gateway_policy: actionGrant("write_gateway_policy", "gateway_policies:write"),
            delete_gateway_policy: actionGrant("delete_gateway_policy", "gateway_policies:delete"),
          },
        },
      ],
    },
  ],
};
