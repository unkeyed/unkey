import { HeartbeatCheck } from "checkly/constructs";
import { incidentIo, slack } from "../alert-channels";

new HeartbeatCheck("agent", {
  alertChannels: [incidentIo],
  name: "Agent",
  activated: true,
  period: 5,
  periodUnit: "minutes",
  grace: 1,
  graceUnit: "minutes",
});

new HeartbeatCheck("workflows-refills", {
  alertChannels: [slack],
  name: "Workflows: Refill",
  activated: true,
  period: 1,
  periodUnit: "days",
  grace: 1,
  graceUnit: "hours",
});

new HeartbeatCheck("workflows-count-keys", {
  alertChannels: [slack],
  name: "Workflows: Count Keys",
  activated: true,
  period: 5,
  periodUnit: "minutes",
  grace: 1,
  graceUnit: "minutes",
});

new HeartbeatCheck("quota-checks", {
  alertChannels: [slack],
  name: "Github Actions: Quota Checks",
  activated: true,
  period: 1,
  periodUnit: "days",
  grace: 1,
  graceUnit: "hours",
});
