import { INSTANCE_TOKEN, type ScopeCatalogue, actionGrant } from "./catalogue.types";

const PROJECT_PATH = `projects/${INSTANCE_TOKEN}`;
const APP_PATH = `${PROJECT_PATH}/apps/*`;
const ENVIRONMENT_PATH = `${APP_PATH}/environments/*`;
const DEPLOYMENT_PATH = `${ENVIRONMENT_PATH}/deployments/*`;
const KEYSPACE_PATH = `${PROJECT_PATH}/keyspaces/*`;
const NAMESPACE_PATH = `${PROJECT_PATH}/ratelimits/namespaces/*`;

export const projectsCatalogue: ScopeCatalogue = {
  scope: "projects",
  label: "Specific projects",
  allLabel: "All projects",
  instanceNoun: "projects",
  groups: [
    {
      id: "project",
      label: "Projects",
      rows: [
        {
          id: "project",
          label: "Projects",
          path: PROJECT_PATH,
          resource: "project",
          actions: {
            read_project: actionGrant("read_project"),
            write_project: actionGrant("write_project"),
            delete_project: actionGrant("delete_project"),
          },
        },
      ],
    },
    {
      id: "apps",
      label: "Apps",
      rows: [
        {
          id: "app",
          label: "Apps",
          path: APP_PATH,
          resource: "app",
          actions: {
            read_app: actionGrant("read_app"),
            write_app: actionGrant("write_app"),
            delete_app: actionGrant("delete_app"),
          },
        },
      ],
    },
    {
      id: "environments",
      label: "Environments",
      rows: [
        {
          id: "environment",
          label: "Environments",
          path: ENVIRONMENT_PATH,
          resource: "environment",
          actions: {
            read_environment: actionGrant("read_environment"),
            write_environment: actionGrant("write_environment"),
            delete_environment: actionGrant("delete_environment"),
          },
        },
        {
          id: "variable",
          label: "Environment variables",
          path: `${ENVIRONMENT_PATH}/variables/*`,
          resource: "variable",
          actions: {
            read_environment_variable: actionGrant("read_environment_variable"),
            write_environment_variable: actionGrant("write_environment_variable"),
            delete_environment_variable: actionGrant("delete_environment_variable"),
          },
        },
        {
          id: "domain",
          label: "Domains",
          path: `${ENVIRONMENT_PATH}/domains/*`,
          resource: "domain",
          actions: {
            read_domain: actionGrant("read_domain"),
            write_domain: actionGrant("write_domain"),
            delete_domain: actionGrant("delete_domain"),
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
          resource: "deployment",
          actions: {
            read_deployment: actionGrant("read_deployment"),
            write_deployment: actionGrant("write_deployment"),
            delete_deployment: actionGrant("delete_deployment"),
          },
        },
        {
          id: "deployment_log",
          label: "Runtime logs",
          path: `${DEPLOYMENT_PATH}/logs`,
          resource: "deployment_log",
          actions: {
            read_deployment_logs: actionGrant("read_deployment_logs"),
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
          resource: "gateway_log",
          actions: {
            read_gateway_logs: actionGrant("read_gateway_logs"),
          },
        },
        {
          id: "gateway_policy",
          label: "Gateway policies",
          path: `${ENVIRONMENT_PATH}/gateway/policies/*`,
          resource: "gateway_policy",
          actions: {
            read_gateway_policy: actionGrant("read_gateway_policy"),
            write_gateway_policy: actionGrant("write_gateway_policy"),
            delete_gateway_policy: actionGrant("delete_gateway_policy"),
          },
        },
      ],
    },
    {
      id: "keyspaces",
      label: "Key management",
      rows: [
        {
          id: "keyspace",
          label: "Keyspaces",
          path: KEYSPACE_PATH,
          resource: "keyspace",
          actions: {
            read_keyspace: actionGrant("read_keyspace"),
            write_keyspace: actionGrant("write_keyspace"),
            delete_keyspace: actionGrant("delete_keyspace"),
          },
        },
        {
          id: "keyspace_log",
          label: "Logs",
          path: `${KEYSPACE_PATH}/logs`,
          resource: "keyspace_log",
          actions: {
            read_keyspace_logs: actionGrant("read_keyspace_logs"),
          },
        },
        {
          id: "key",
          label: "Keys",
          path: `${KEYSPACE_PATH}/keys/*`,
          resource: "key",
          actions: {
            read_key: actionGrant("read_key"),
            write_key: actionGrant("write_key"),
            delete_key: actionGrant("delete_key"),
            decrypt_key: actionGrant("decrypt_key"),
            verify_key: actionGrant("verify_key"),
          },
        },
      ],
    },
    {
      id: "ratelimits",
      label: "Rate limiting",
      rows: [
        {
          id: "ratelimit_namespace",
          label: "Rate limit namespaces",
          path: NAMESPACE_PATH,
          resource: "ratelimit_namespace",
          actions: {
            read_ratelimit_namespace: actionGrant("read_ratelimit_namespace"),
            write_ratelimit_namespace: actionGrant("write_ratelimit_namespace"),
            delete_ratelimit_namespace: actionGrant("delete_ratelimit_namespace"),
            limit_ratelimit_namespace: actionGrant("limit_ratelimit_namespace"),
          },
        },
        {
          id: "ratelimit_log",
          label: "Logs",
          path: `${NAMESPACE_PATH}/logs`,
          resource: "ratelimit_log",
          actions: {
            read_ratelimit_logs: actionGrant("read_ratelimit_logs"),
          },
        },
        {
          id: "ratelimit_override",
          label: "Rate limit overrides",
          path: `${NAMESPACE_PATH}/overrides/*`,
          resource: "ratelimit_override",
          actions: {
            read_ratelimit_override: actionGrant("read_ratelimit_override"),
            write_ratelimit_override: actionGrant("write_ratelimit_override"),
            delete_ratelimit_override: actionGrant("delete_ratelimit_override"),
          },
        },
      ],
    },
    {
      id: "access",
      label: "Identity and RBAC",
      rows: [
        {
          id: "identity",
          label: "Identities",
          path: `${PROJECT_PATH}/identities/*`,
          resource: "identity",
          actions: {
            read_identity: actionGrant("read_identity"),
            write_identity: actionGrant("write_identity"),
            delete_identity: actionGrant("delete_identity"),
          },
        },
        {
          id: "role",
          label: "Roles",
          path: `${PROJECT_PATH}/rbac/roles/*`,
          resource: "role",
          actions: {
            read_role: actionGrant("read_role"),
            write_role: actionGrant("write_role"),
            delete_role: actionGrant("delete_role"),
          },
        },
        {
          id: "permission",
          label: "Permissions",
          path: `${PROJECT_PATH}/rbac/permissions/*`,
          resource: "permission",
          actions: {
            read_permission: actionGrant("read_permission"),
            write_permission: actionGrant("write_permission"),
            delete_permission: actionGrant("delete_permission"),
          },
        },
      ],
    },
  ],
};
