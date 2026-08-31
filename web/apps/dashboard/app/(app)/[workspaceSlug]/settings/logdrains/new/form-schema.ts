import { z } from "zod";
import { headerFieldsSchema } from "../header-fields";

export type Kind = "http" | "axiom";
export type HttpFormat = "json" | "ndjson";

const httpsUrlSchema = z
  .string()
  .url("Enter a valid URL")
  .superRefine((value, context) => {
    if (!URL.canParse(value)) {
      return;
    }
    const url = new URL(value);
    if (url.protocol !== "https:") {
      context.addIssue({ code: "custom", message: "URL must use HTTPS" });
    }
    if (url.username !== "" || url.password !== "") {
      context.addIssue({ code: "custom", message: "URL must not contain credentials" });
    }
  });

export const formSchema = z
  .object({
    kind: z.enum(["http", "axiom"]),
    name: z.string().trim().min(1, "Enter a name").max(128, "Name must be 128 characters or less"),
    url: z.string(),
    format: z.enum(["json", "ndjson"]),
    headers: headerFieldsSchema,
    dataset: z.string(),
    token: z.string(),
    startFrom: z.enum(["now", "beginning"]),
  })
  .superRefine((values, context) => {
    switch (values.kind) {
      case "http": {
        const url = httpsUrlSchema.safeParse(values.url);
        if (!url.success) {
          context.addIssue({
            code: "custom",
            path: ["url"],
            message: url.error.issues[0]?.message ?? "Enter a valid HTTPS URL",
          });
        }
        break;
      }
      case "axiom":
        if (values.dataset.trim() === "") {
          context.addIssue({ code: "custom", path: ["dataset"], message: "Enter a dataset" });
        }
        if (values.token.trim() === "") {
          context.addIssue({ code: "custom", path: ["token"], message: "Enter a token" });
        }
        break;
      default:
        throw new Error(`Unsupported log drain sink: ${values.kind satisfies never}`);
    }
  });

export type FormValues = z.infer<typeof formSchema>;
