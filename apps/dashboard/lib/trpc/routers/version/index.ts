import { t } from "../../trpc";
import { listVersions } from "./list";

export const versionRouter = t.router({
  list: listVersions,
});
