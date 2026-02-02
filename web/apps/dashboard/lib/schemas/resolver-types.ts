import type { FieldValues, Resolver } from "react-hook-form";
import type { z } from "zod";

/**
 * Helper type to create a properly typed resolver for discriminated union schemas.
 *
 * This is needed because Zod v4's discriminated unions create complex union types
 * that react-hook-form's zodResolver cannot properly infer. This helper extracts
 * the inferred type and creates a compatible resolver type.
 *
 * @example
 * ```typescript
 * const methods = useForm<z.infer<typeof mySchema>>({
 *   resolver: zodResolver(mySchema) as DiscriminatedUnionResolver<typeof mySchema>,
 * });
 * ```
 */
export type DiscriminatedUnionResolver<T extends z.ZodTypeAny> = Resolver<
  z.infer<T> extends FieldValues ? z.infer<T> : never
>;
