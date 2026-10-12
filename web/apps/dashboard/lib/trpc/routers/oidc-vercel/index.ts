import { t } from "../../trpc";
import { createOIDCVercelIntegration } from "./create";
import { deleteOIDCVercelIntegration } from "./delete";
import { listOIDCVercelIntegrations } from "./list";
import { updateOIDCVercelIntegration } from "./update";

export const oidcVercel = t.router({
  create: createOIDCVercelIntegration,
  delete: deleteOIDCVercelIntegration,
  list: listOIDCVercelIntegrations,
  update: updateOIDCVercelIntegration,
});
