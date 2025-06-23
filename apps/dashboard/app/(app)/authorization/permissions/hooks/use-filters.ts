import { createUseFilters } from "@/lib/filter-builders";
import { permissionsFilter } from "../filters.schema";

export const useFilters = createUseFilters(permissionsFilter);
