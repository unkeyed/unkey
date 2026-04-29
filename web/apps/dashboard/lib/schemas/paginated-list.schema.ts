import { z } from "zod";

const DEFAULT_MAX_LIMIT = 100;

// Builds the zod schema for a server-paginated list query payload used by
// tRPC inputs (e.g. roles/permissions query routers). The shape mirrors what
// `usePaginatedListQuery` sends: per-field filter arrays plus page/limit/
// sortBy/sortOrder. Kept generic so each feature supplies its own operator
// enum, filter field names, and sort enum, and gets a narrowly typed schema
// back.
export function createPaginatedListQueryPayload<
  TOperator extends string,
  TField extends string,
  TSortField extends string,
>(args: {
  operatorEnum: z.ZodType<TOperator>;
  filterFieldNames: readonly TField[];
  sortByEnum: z.ZodType<TSortField>;
  defaultSortField: TSortField;
  maxLimit?: number;
}) {
  const {
    operatorEnum,
    filterFieldNames,
    sortByEnum,
    defaultSortField,
    maxLimit = DEFAULT_MAX_LIMIT,
  } = args;

  const filterItem = z.object({
    operator: operatorEnum,
    value: z.string(),
  });
  const filterArray = z.array(filterItem).nullish();

  const filterFields = filterFieldNames.reduce(
    (acc, name) => {
      acc[name] = filterArray;
      return acc;
    },
    {} as Record<TField, typeof filterArray>,
  );

  return z.object(filterFields).extend({
    page: z.number().int().min(1).optional().default(1),
    limit: z.number().int().min(1).max(maxLimit).optional(),
    sortBy: sortByEnum.optional().default(defaultSortField),
    sortOrder: z.enum(["asc", "desc"]).optional().default("desc"),
  });
}
