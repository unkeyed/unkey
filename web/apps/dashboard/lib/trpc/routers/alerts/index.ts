import { t } from "../../trpc";
import { getAlert } from "./get";
import { listAlerts } from "./list";
import { resolveAlert } from "./resolve";
import { getAlertsSummary } from "./summary";
import { getAlertTimeseries } from "./timeseries";

export const alerts = t.router({
  list: listAlerts,
  get: getAlert,
  resolve: resolveAlert,
  summary: getAlertsSummary,
  timeseries: getAlertTimeseries,
});
