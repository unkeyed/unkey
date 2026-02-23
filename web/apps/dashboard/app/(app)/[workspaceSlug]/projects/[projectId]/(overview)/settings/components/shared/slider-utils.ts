type SliderOption = { readonly label: string; readonly value: number };

export function valueToIndex<T extends readonly SliderOption[]>(options: T, value: number): number {
  const idx = options.findIndex((o) => o.value === value);
  return idx >= 0 ? idx : 0;
}

export function indexToValue<T extends readonly SliderOption[]>(
  options: T,
  index: number,
  fallback: number,
): number {
  return options[index]?.value ?? fallback;
}
