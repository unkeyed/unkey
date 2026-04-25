import { z } from "zod";

export const customDomainSchema = z.object({
  environmentId: z.string().min(1, "Environment is required"),
  domain: z
    .string()
    .min(1, "Domain is required")
    .regex(/^(?!:\/\/)([a-zA-Z0-9-_]+\.)+[a-zA-Z]{2,}$/, "Invalid domain format"),
  alsoAddWww: z.boolean(),
});

export type CustomDomainFormValues = z.infer<typeof customDomainSchema>;
