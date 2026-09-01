import { t } from "../../trpc";
import { createLogdrain } from "./create";
import { deleteLogdrain } from "./delete";
import { getLogdrain } from "./get";
import { listLogdrains } from "./list";
import { getLogdrainMetrics } from "./metrics";
import { getRecentLogdrainErrors } from "./recent-errors";
import { updateLogdrain } from "./update";

export const logdrain = t.router({
  create: createLogdrain,
  get: getLogdrain,
  update: updateLogdrain,
  delete: deleteLogdrain,
  list: listLogdrains,
  metrics: getLogdrainMetrics,
  recentErrors: getRecentLogdrainErrors,
});
