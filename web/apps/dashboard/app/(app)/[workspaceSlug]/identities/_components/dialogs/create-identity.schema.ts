import { z } from "zod";
import { identityExternalIdSchema } from "@/lib/schemas/identity";
import { identityMetadataSchema } from "@/lib/schemas/metadata";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";

export const formSchema = z
  .object({
    externalId: identityExternalIdSchema,
  })
  .extend(identityMetadataSchema.shape)
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
