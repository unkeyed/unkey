import { createUseFilters } from "@/lib/filters/filter-hook";
import { logsSchema } from "../filters.schema";

export const useFilters = createUseFilters(logsSchema);
