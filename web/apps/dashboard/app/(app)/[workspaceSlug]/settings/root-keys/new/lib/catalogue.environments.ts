import { INSTANCE_TOKEN, type ScopeCatalogue } from "./catalogue.types";

const ENVIRONMENT_PATH = `projects/*/apps/*/environments/${INSTANCE_TOKEN}`;

export const environmentsCatalogue: ScopeCatalogue = {
  scope: "environments",
  label: "Environments",
  allLabel: "All environments",
  instanceNoun: "environments",
  groups: [
    {
      id: "environment",
      label: "Environment",
      rows: [
        {
          id: "environment",
          label: "Environment settings",
          path: ENVIRONMENT_PATH,
          resource: "environment",
          actions: {
            write: [{ name: "update_environment" }],
          },
        },
        {
          id: "deployment",
          label: "Deployments",
          path: `${ENVIRONMENT_PATH}/deployments/*`,
          resource: "deployment",
          actions: {
            write: [
              { name: "create_deployment" },
              { name: "start_deployment" },
              { name: "stop_deployment" },
              { name: "promote_deployment" },
              { name: "rollback_deployment" },
            ],
          },
        },
        {
          id: "domain",
          label: "Custom domains",
          path: `${ENVIRONMENT_PATH}/domains/*`,
          resource: "domain",
          actions: {
            write: [
              { name: "create_domain", path: ENVIRONMENT_PATH },
              { name: "verify_domain", path: ENVIRONMENT_PATH },
            ],
          },
        },
        {
          id: "variable",
          label: "Environment variables",
          path: ENVIRONMENT_PATH,
          resource: "variable",
          actions: {
            read: [{ name: "read_environment_variables" }],
            write: [{ name: "create_variable" }, { name: "set_environment_variables" }],
            delete: [{ name: "remove_environment_variables" }],
          },
        },
      ],
    },
  ],
};
