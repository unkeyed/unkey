export type Env = {
  // Add your bindings here, e.g. Workers KV, D1, Workers AI, etc.
  REFILL_REMAINING: Workflow;
  COUNT_KEYS: Workflow;

  HEARTBEAT_URL_REFILLS: string;
  HEARTBEAT_URL_COUNT_KEYS: string;
  DATABASE_HOST: string;
  DATABASE_USERNAME: string;
  DATABASE_PASSWORD: string;
};
