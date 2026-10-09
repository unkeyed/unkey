import { z } from "zod";

const HOSTNAME = /^(?:[^\s.*:/?#@%\\[\]]+\.)+[^\s.*:/?#@%\\[\]]{2,}$/;

export const customDomainSchema = z.object({
  environmentId: z.string().min(1, "Environment is required"),
  domain: z
    .string()
    .trim()
    .toLowerCase()
    .min(1, "Domain is required")
    .pipe(z.string().regex(HOSTNAME, "Enter a domain like api.example.com")),
});

export type CustomDomainFormValues = z.infer<typeof customDomainSchema>;
