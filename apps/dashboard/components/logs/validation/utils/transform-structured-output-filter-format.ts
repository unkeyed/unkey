export const transformStructuredOutputToFilters = <
  TField extends string,
  TOperator extends string,
  TValue extends string | number,
>(
  data: {
    filters: Array<{
      field: TField;
      filters: Array<{ operator: TOperator; value: TValue }>;
    }>;
  },
  existingFilters: Array<{
    id: string;
    field: TField;
    operator: TOperator;
    value: TValue;
  }> = [],
): Array<{ id: string; field: TField; operator: TOperator; value: TValue }> => {
  const uniqueFilters = [...existingFilters];
  const seenFilters = new Set(existingFilters.map((f) => `${f.field}-${f.operator}-${f.value}`));

  for (const filterGroup of data.filters) {
    filterGroup.filters.forEach((filter) => {
      const baseFilter = {
        field: filterGroup.field,
        operator: filter.operator,
        value: filter.value,
      };

      const filterKey = `${baseFilter.field}-${baseFilter.operator}-${baseFilter.value}`;
      if (seenFilters.has(filterKey)) {
        return;
      }

      uniqueFilters.push({
        id: crypto.randomUUID(),
        ...baseFilter,
      });
      seenFilters.add(filterKey);
    });
  }

  return uniqueFilters;
};
