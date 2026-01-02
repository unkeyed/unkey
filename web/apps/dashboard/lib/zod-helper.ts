import { z } from "zod";

export const formSchema = z.object({
  bytes: z.coerce.number({
    errorMap: (issue, { defaultError }) => ({
      message:
        issue.code === "invalid_type" ? "Amount must be a number and greater than 0" : defaultError,
    }),
  }),
  prefix: z
    .string()
    .max(16, { message: "Please limit the prefix to under 16 characters." })
    .optional(),
  ownerId: z.string().optional(),
  name: z.string().optional(),
  metaEnabled: z.boolean(),
  meta: z.string().optional(),
  limitEnabled: z.boolean(),
  limit: z
    .object({
      remaining: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type"
                ? "Remaining amount must be greater than 0"
                : defaultError,
          }),
        })
        .int()
        .positive({ message: "Please enter a positive number" }),
      refill: z
        .object({
          interval: z.enum(["none", "daily", "monthly"]),
          amount: z.coerce
            .number({
              errorMap: (issue, { defaultError }) => ({
                message:
                  issue.code === "invalid_type"
                    ? "Refill amount must be greater than 0 and a integer"
                    : defaultError,
              }),
            })
            .int()
            .min(1)
            .positive(),
          refillDay: z.coerce
            .number({
              errorMap: (issue, { defaultError }) => ({
                message:
                  issue.code === "invalid_type"
                    ? "Refill day must be an integer between 1 and 31"
                    : defaultError,
              }),
            })
            .int()
            .min(1)
            .max(31)
            .optional(),
        })
        .optional(),
    })
    .optional(),
  expireEnabled: z.boolean(),
  expires: z.coerce
    .date()
    .min(new Date(Date.now() + 2 * 60000))
    .optional(),
  ratelimitEnabled: z.boolean(),
  ratelimit: z
    .object({
      type: z.enum(["consistent", "fast"]).default("fast"),
      refillInterval: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type"
                ? "Refill interval must be greater than 0"
                : defaultError,
          }),
        })
        .positive({ message: "Refill interval must be greater than 0" }),
      refillRate: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type" ? "Refill rate must be greater than 0" : defaultError,
          }),
        })
        .positive({ message: "Refill rate must be greater than 0" }),
      limit: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type" ? "Refill limit must be greater than 0" : defaultError,
          }),
        })
        .positive({ message: "Limit must be greater than 0" }),
    })
    .optional(),
});
