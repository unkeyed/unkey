import { z } from "zod";

export const rootKeySchema = z.object({
  name: z.string().trim().min(1, "Give this key a name."),
});

export type RootKeyFormValues = z.infer<typeof rootKeySchema>;

export const rootKeyDefaultValues: RootKeyFormValues = {
  name: "",
};
