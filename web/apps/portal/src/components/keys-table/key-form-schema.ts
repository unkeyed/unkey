import { z } from "zod";

export const keyFormSchema = z.object({
  name: z.string().trim().max(255, "Name must be 255 characters or less."),
  expiration: z
    .date()
    .refine((d) => d.getTime() > Date.now(), {
      message: "Expiration must be in the future.",
    })
    .optional(),
});

export type KeyFormValues = z.infer<typeof keyFormSchema>;
