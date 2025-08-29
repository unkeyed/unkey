import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const envVarSchema = z.object({
  id: z.string(),
  key: z.string(),
  value: z.string(),
  isSecret: z.boolean(),
});

const environmentVariablesOutputSchema = z.object({
  production: z.array(envVarSchema),
  preview: z.array(envVarSchema),
  development: z.array(envVarSchema).optional(),
});

export type EnvironmentVariables = z.infer<typeof environmentVariablesOutputSchema>;
export type EnvVar = z.infer<typeof envVarSchema>;

const VARIABLES: EnvironmentVariables = {
  production: [
    {
      id: "1",
      key: "DATABASE_URL",
      value: "postgresql://user:pass@prod.db.com:5432/app",
      isSecret: true,
    },
    {
      id: "2",
      key: "API_KEY",
      value: "sk_prod_1234567890abcdef",
      isSecret: true,
    },
    {
      id: "3",
      key: "NODE_ENV",
      value: "production",
      isSecret: false,
    },
    {
      id: "4",
      key: "REDIS_URL",
      value: "redis://prod.redis.com:6379",
      isSecret: true,
    },
    {
      id: "5",
      key: "LOG_LEVEL",
      value: "info",
      isSecret: false,
    },
  ],
  preview: [
    {
      id: "6",
      key: "DATABASE_URL",
      value: "postgresql://user:pass@staging.db.com:5432/app",
      isSecret: true,
    },
    {
      id: "7",
      key: "API_KEY",
      value: "sk_test_abcdef1234567890",
      isSecret: true,
    },
    {
      id: "8",
      key: "NODE_ENV",
      value: "development",
      isSecret: false,
    },
  ],
};

export const getEnvs = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .output(environmentVariablesOutputSchema)
  .query(() => {
    return VARIABLES;
  });
