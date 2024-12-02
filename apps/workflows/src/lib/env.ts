export type Env = {
  // Add your bindings here, e.g. Workers KV, D1, Workers AI, etc.
  REFILL_REMAINING: Workflow;

  HEARTBEAT_URL_REFILLS: string;
  DATABASE_HOST: string;
  DATABASE_USERNAME: string;
  DATABASE_PASSWORD: string;
};
