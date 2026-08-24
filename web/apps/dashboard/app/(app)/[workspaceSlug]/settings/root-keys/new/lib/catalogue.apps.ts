import { INSTANCE_TOKEN, type ScopeCatalogue } from "./catalogue.types";

const APP_PATH = `projects/*/apps/${INSTANCE_TOKEN}`;
const ENVIRONMENT_PATH = `${APP_PATH}/environments/*`;

export const appsCatalogue: ScopeCatalogue = {
  scope: "apps",
  label: "Apps",
  allLabel: "All apps",
  instanceNoun: "apps",
  groups: [
    {
      id: "app",
      label: "App",
      rows: [
        {
          id: "app",
          label: "App settings",
          path: APP_PATH,
          resource: "app",
          actions: {
            write: [{ name: "update_app" }, { name: "connect_repository" }],
          },
        },
        {
          id: "environment",
          label: "Environments",
          path: ENVIRONMENT_PATH,
          resource: "environment",
          actions: {
            write: [{ name: "create_environment", path: APP_PATH }, { name: "update_environment" }],
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
