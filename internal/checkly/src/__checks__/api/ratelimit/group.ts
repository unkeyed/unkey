import { CheckGroup } from "checkly/constructs";
import { ALL_LOCATIONS } from "../../../locations";

export const ratelimitsV1 = new CheckGroup("/v1/ratelimits", {
  name: "ratelimits",
  locations: ALL_LOCATIONS,
});
