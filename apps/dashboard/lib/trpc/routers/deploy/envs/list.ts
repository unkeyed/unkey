import { envVarSchema } from "@/app/(app)/projects/[projectId]/details/env-variables-section/types";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const environmentVariablesOutputSchema = z.object({
  production: z.array(envVarSchema),
  preview: z.array(envVarSchema),
});

export type EnvironmentVariables = z.infer<typeof environmentVariablesOutputSchema>;
export type EnvVar = z.infer<typeof envVarSchema>;

export const VARIABLES: EnvironmentVariables = {
  production: [
    {
      id: "1",
      key: "DATABASE_URL",
      value: "postgresql://user:pass@prod.db.com:5432/app",
      type: "secret",
    },
    {
      id: "2",
      key: "API_KEY",
      value: "sk_prod_1234567890abcdef",
      type: "secret",
    },
    {
      id: "3",
      key: "NODE_ENV",
      value: "production",
      type: "env",
    },
    {
      id: "4",
      key: "REDIS_URL",
      value: "redis://prod.redis.com:6379",
      type: "secret",
    },
    {
      id: "5",
      key: "LOG_LEVEL",
      value: "info",
      type: "env",
    },
  ],
  preview: [
    {
      id: "6",
      key: "DATABASE_URL",
      value: "postgresql://user:pass@staging.db.com:5432/app",
      type: "secret",
    },
    {
      id: "7",
      key: "API_KEY",
      value: "sk_test_abcdef1234567890",
      type: "secret",
    },
    {
      id: "8",
      key: "NODE_ENV",
      value: "development",
      type: "env",
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
