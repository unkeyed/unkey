import { type ClassValue, clsx } from "cn";

type ClassProp =
  | { class: ClassValue; className?: never }
  | { class?: never; className: ClassValue }
  | { class?: never; className?: never };

type StringToBoolean<T> = T extends "true" | "false" ? boolean : T;

type VariantSchema = Record<string, Record<string, ClassValue>>;

type VariantSelection<T extends VariantSchema> = {
  [Variant in keyof T]?: StringToBoolean<keyof T[Variant]> | null | undefined;
};

type CompoundVariantSelection<T extends VariantSchema> = {
  [Variant in keyof T]?:
    | StringToBoolean<keyof T[Variant]>
    | StringToBoolean<keyof T[Variant]>[]
    | undefined;
};

type VariantConfig<T> = T extends VariantSchema
  ? {
      variants?: T;
      defaultVariants?: VariantSelection<T>;
      compoundVariants?: ((VariantSelection<T> | CompoundVariantSelection<T>) & ClassProp)[];
    }
  : never;

type VariantInput<T> = T extends VariantSchema ? VariantSelection<T> & ClassProp : ClassProp;

export type VariantProps<Component extends (...args: never) => unknown> = Omit<
  Exclude<Parameters<Component>[0], undefined>,
  "class" | "className"
>;

const variantKey = (value: unknown) =>
  typeof value === "boolean" ? `${value}` : value === 0 ? "0" : value;

export const cva =
  <T>(base?: ClassValue, config?: VariantConfig<T>) =>
  (props?: VariantInput<T>): string => {
    const input: Record<string, unknown> = props ?? {};
    if (config?.variants == null) {
      return clsx(base, props?.class, props?.className);
    }
    const variants: VariantSchema = config.variants;
    const defaults: Record<string, unknown> = config.defaultVariants ?? {};

    const variantClasses = Object.keys(variants).map((variant) => {
      const selected = input[variant];
      if (selected === null) {
        return null;
      }
      const key = variantKey(selected) || variantKey(defaults[variant]);
      return variants[variant][String(key)];
    });

    const resolved: Record<string, unknown> = { ...defaults };
    for (const [key, value] of Object.entries(input)) {
      if (value !== undefined) {
        resolved[key] = value;
      }
    }
    const compoundClasses = (config.compoundVariants ?? []).flatMap(
      ({ class: compoundClass, className: compoundClassName, ...conditions }) =>
        Object.entries(conditions).every(([key, expected]) =>
          Array.isArray(expected) ? expected.includes(resolved[key]) : resolved[key] === expected,
        )
          ? [compoundClass, compoundClassName]
          : [],
    );

    return clsx(base, variantClasses, compoundClasses, props?.class, props?.className);
  };
