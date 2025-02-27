export type Env = {
  // Add your bindings here, e.g. Workers KV, D1, Workers AI, etc.
  REFILL_REMAINING: Workflow;
  COUNT_KEYS: Workflow;
  INVOICING: Workflow;

  HEARTBEAT_URL_REFILLS: string;
  HEARTBEAT_URL_COUNT_KEYS: string;
  DATABASE_HOST: string;
  DATABASE_USERNAME: string;
  DATABASE_PASSWORD: string;

  CLICKHOUSE_URL: string;

  STRIPE_SECRET_KEY: string;

  STRIPE_PRODUCT_ID_KEY_VERIFICATIONS: string;
  STRIPE_PRODUCT_ID_RATELIMITS: string;
  STRIPE_PRODUCT_ID_PRO_PLAN: string;
  STRIPE_PRODUCT_ID_SUPPORT: string;
};
