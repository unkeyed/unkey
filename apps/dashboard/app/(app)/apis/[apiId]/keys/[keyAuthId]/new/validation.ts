import { z } from "zod";

export const formSchema = z.object({
  bytes: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type"
            ? "Amount must be a number and greater than 0"
            : defaultError,
      }),
    })
    .default(16),
  prefix: z
    .string()
    .trim()
    .max(8, { message: "Please limit the prefix to under 8 characters." })
    .optional(),
  ownerId: z.string().trim().optional(),
  name: z.string().trim().optional(),
  metaEnabled: z.boolean().default(false),
  meta: z
    .string()
    .refine(
      (s) => {
        try {
          JSON.parse(s);
          return true;
        } catch {
          return false;
        }
      },
      {
        message: "Must be valid json",
      },
    )
    .optional(),
  limitEnabled: z.boolean().default(false),
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
          amount: z
            .literal("")
            .transform(() => undefined)
            .or(
              z.coerce
                .number({
                  errorMap: (issue, { defaultError }) => ({
                    message:
                      issue.code === "invalid_type"
                        ? "Refill amount must be greater than 0 and a integer"
                        : defaultError,
                  }),
                })
                .int()
                .positive(),
            ),
          refillDay: z
            .literal("")
            .transform(() => undefined)
            .or(
              z.coerce
                .number({
                  errorMap: (issue, { defaultError }) => ({
                    message:
                      issue.code === "invalid_type"
                        ? "Refill day must be an integer between 1 and 31"
                        : defaultError,
                  }),
                })
                .int()
                .max(31)
                .positive(),
            )
            .optional(),
        })
        .optional(),
    })
    .optional(),
  expireEnabled: z.boolean().default(false),
  expires: z.coerce
    .date()
    .min(new Date(new Date().getTime() + 2 * 60000))
    .optional(),
  ratelimitEnabled: z.boolean().default(false),
  ratelimit: z
    .object({
      async: z.boolean().default(false),
      duration: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type" ? "Duration must be greater than 0" : defaultError,
          }),
        })
        .positive({ message: "Refill interval must be greater than 0" }),
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
  environment: z.string().optional(),
});
