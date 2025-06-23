import { createUseFilters } from "@/lib/filter-builders";
import { logsFilter } from "../filters.schema";

export const useFilters = createUseFilters(logsFilter);
