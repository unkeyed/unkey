import { HeartbeatCheck } from "checkly/constructs";
import { incidentIo } from "../alert-channels";

new HeartbeatCheck("vault", {
  alertChannels: [incidentIo],
  name: "Vault",
  activated: true,
  period: 5,
  periodUnit: "minutes",
  grace: 1,
  graceUnit: "minutes",
});

new HeartbeatCheck("agent", {
  alertChannels: [incidentIo],
  name: "Agent",
  activated: true,
  period: 5,
  periodUnit: "minutes",
  grace: 1,
  graceUnit: "minutes",
});
