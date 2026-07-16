import { identityExternalIdSchema } from "@/lib/schemas/identity";
import { metadataSchema } from "@/lib/schemas/metadata";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";
import { z } from "zod";

export const formSchema = z
  .object({
    externalId: identityExternalIdSchema,
  })
  .extend(metadataSchema.shape)
  .extend(ratelimitSchema.shape);

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
