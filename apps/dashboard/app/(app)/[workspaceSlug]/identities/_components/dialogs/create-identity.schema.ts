import { metadataSchema } from "@/lib/schemas/metadata";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";
import { z } from "zod";

export const formSchema = z
  .object({
    externalId: z
      .string()
      .transform((s) => s.trim())
      .refine((trimmed) => trimmed.length >= 3, "External ID must be at least 3 characters")
      .refine((trimmed) => trimmed.length <= 255, "External ID must be 255 characters or fewer")
      .refine((trimmed) => trimmed !== "", "External ID cannot be only whitespace"),
  })
  .merge(metadataSchema)
  .merge(ratelimitSchema);

export type FormValues = z.infer<typeof formSchema>;

export const getDefaultValues = (): FormValues => ({
  externalId: "",
  metadata: {
    enabled: false,
  },
  ratelimit: {
    enabled: false,
    data: [
      {
        name: "default",
        limit: 10,
        refillInterval: 1000,
        autoApply: true,
      },
    ],
  },
});
