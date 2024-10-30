import { z } from "zod";

// note that this doesn't include the react icons, so that we can load the typescript .ts file into our content-collection config
export const categories = [
  {
    slug: "api-specification",
    title: "API Specification",
    description:
      "API & Web standards for defining data formats and interactions (e.g. OpenAPI, REST, HTTP Requests, etc.)",
  },
] as const;

// Extract slug values to create a union type
type CategorySlug = (typeof categories)[number]["slug"];

// Create a Zod enum from the CategorySlug type
export const categoryEnum = z.enum(
  categories.map((c) => c.slug) as [CategorySlug, ...Array<CategorySlug>],
);

export type CategoryEnum = z.infer<typeof categoryEnum>;
