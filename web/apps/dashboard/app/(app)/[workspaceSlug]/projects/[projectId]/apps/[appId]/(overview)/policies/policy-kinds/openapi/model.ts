import { IconFileSettingsOutline18 } from "@unkey/icons";
import { z } from "zod";
import { sharedFormFields } from "../shared";
import type { PolicyKindModel } from "../types";

export const openapiFormSchema = z.object({
  ...sharedFormFields,
  type: z.literal("openapi"),
});

type OpenapiFormValues = z.infer<typeof openapiFormSchema>;

export const openapi: PolicyKindModel<"openapi", OpenapiFormValues> = {
  label: "OpenAPI Validation",
  Icon: IconFileSettingsOutline18,
  does: "Checks the request against your OpenAPI spec.",
  rejects: [{ status: 400, reason: "The request does not match your spec" }],
  defaults: () => ({ type: "openapi" }),
  toWire: () => ({ type: "openapi", openapi: {} }),
  fromWire: () => ({ type: "openapi" }),
};

export const OPENAPI_SUMMARY = "Auto-scraped spec";
