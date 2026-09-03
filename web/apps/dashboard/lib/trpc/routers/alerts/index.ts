import { t } from "../../trpc";
import { listAlertDeployments } from "./deployments";
import { getAlert } from "./get";
import { listAlerts } from "./list";
import { getAlertSeries } from "./series";
import { getAlertsSummary } from "./summary";

export const alerts = t.router({
  list: listAlerts,
  get: getAlert,
  summary: getAlertsSummary,
  series: getAlertSeries,
  deployments: listAlertDeployments,
});
