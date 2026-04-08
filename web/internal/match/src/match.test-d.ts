import { describe, expectTypeOf, it } from "vitest";
import { P, match } from "./index";

describe("type: discriminated union narrowing", () => {
  type Shape = { kind: "circle"; r: number } | { kind: "rect"; w: number; h: number };

  it("narrows handler param to matched variant", () => {
    const shape = {} as Shape;
    match(shape)
      .with({ kind: "circle" }, (s) => {
        expectTypeOf(s).toEqualTypeOf<{ kind: "circle"; r: number }>();
        return 0;
      })
      .with({ kind: "rect" }, (s) => {
        expectTypeOf(s).toEqualTypeOf<{ kind: "rect"; w: number; h: number }>();
        return 0;
      })
      .exhaustive();
  });

  it("exhaustive() compiles when all variants are covered", () => {
    const shape = {} as Shape;
    const result = match(shape)
      .with({ kind: "circle" }, () => 1)
      .with({ kind: "rect" }, () => 2)
      .exhaustive();
    expectTypeOf(result).toEqualTypeOf<number>();
  });

  it("otherwise() receives the remaining unmatched type", () => {
    const shape = {} as Shape;
    match(shape)
      .with({ kind: "circle" }, () => 1)
      .otherwise((remaining) => {
        expectTypeOf(remaining).toEqualTypeOf<{
          kind: "rect";
          w: number;
          h: number;
        }>();
        return 2;
      });
  });
});

describe("type: literal narrowing", () => {
  it("narrows string union to matched literal", () => {
    const v = {} as "a" | "b" | "c";
    match(v)
      .with("a", (s) => {
        expectTypeOf(s).toEqualTypeOf<"a">();
        return 0;
      })
      .with("b", (s) => {
        expectTypeOf(s).toEqualTypeOf<"b">();
        return 0;
      })
      .with("c", () => 0)
      .exhaustive();
  });

  it("does not exhaust base string type with a single literal", () => {
    const v = {} as string;
    const builder = match(v).with("a", () => 1);
    // TRemaining should still be string, so otherwise is needed
    builder.otherwise(() => 2);
  });
});

describe("type: P helpers", () => {
  it("P.string narrows to string", () => {
    const v = {} as string | number;
    match(v)
      .with(P.string, (s) => {
        expectTypeOf(s).toEqualTypeOf<string>();
        return 0;
      })
      .with(P.number, (n) => {
        expectTypeOf(n).toEqualTypeOf<number>();
        return 0;
      })
      .exhaustive();
  });

  it("P.nullish narrows to null | undefined", () => {
    const v = {} as string | null | undefined;
    match(v)
      .with(P.nullish, (s) => {
        expectTypeOf(s).toEqualTypeOf<null | undefined>();
        return 0;
      })
      .otherwise(() => 1);
  });

  it("P.nonNullable narrows out null/undefined", () => {
    const v = {} as string | null;
    match(v)
      .with(P.nonNullable, (s) => {
        expectTypeOf(s).toEqualTypeOf<string>();
        return 0;
      })
      .otherwise(() => 1);
  });

  it("P._ matches unknown (wildcard)", () => {
    const v = {} as string | number;
    match(v)
      .with(P._, (s) => {
        expectTypeOf(s).toEqualTypeOf<string | number>();
        return 0;
      })
      .otherwise(() => 1);
  });
});

describe("type: object patterns with P helpers", () => {
  it("narrows object field via P.string", () => {
    type Data = { error: string | null; ok: boolean };
    const data = {} as Data;
    match(data)
      .with({ error: P.string }, (d) => {
        expectTypeOf(d).toEqualTypeOf<{ error: string; ok: boolean }>();
        return 0;
      })
      .otherwise(() => 1);
  });
});

describe("type: Pattern<T> autocomplete surface", () => {
  it("accepts literal union members", () => {
    const v = {} as "a" | "b" | "c";
    match(v)
      .with("a", () => 1)
      .with("b", () => 2)
      .with("c", () => 3)
      .exhaustive();
  });

  it("rejects literals not in the union", () => {
    const v = {} as "a" | "b";
    match(v)
      // @ts-expect-error "c" is not assignable to Pattern<"a" | "b">
      .with("c", () => 1)
      .otherwise(() => 0);
  });

  it("partial object patterns are allowed", () => {
    const v = {} as { a: boolean; b: boolean };
    match(v)
      .with({ a: true }, () => 1)
      .otherwise(() => 0);
  });
});

describe("type: multi-pattern OR", () => {
  it("handler receives union of both narrowed types", () => {
    const v = {} as "a" | "b" | "c";
    match(v)
      .with("a", "b", (s) => {
        expectTypeOf(s).toEqualTypeOf<"a" | "b">();
        return 0;
      })
      .with("c", () => 0)
      .exhaustive();
  });
});

describe("type: guard overload", () => {
  it("guard and handler both receive narrowed type", () => {
    const v = {} as number;
    match(v)
      .with(
        P.number,
        (n) => {
          expectTypeOf(n).toEqualTypeOf<number>();
          return n > 0;
        },
        (n) => {
          expectTypeOf(n).toEqualTypeOf<number>();
          return "positive";
        },
      )
      .otherwise(() => "other");
  });
});

describe("type: .returnType() constraint", () => {
  it("constrains handler return types", () => {
    const v = {} as "a" | "b";
    const result = match(v)
      .returnType<number>()
      .with("a", () => 1)
      .with("b", () => 2)
      .exhaustive();
    expectTypeOf(result).toEqualTypeOf<number>();
  });
});

describe("type: .when()", () => {
  it("handler receives narrowed type for type guard predicate", () => {
    const v = {} as string | number;
    match(v)
      .when(
        (x): x is string => typeof x === "string",
        (x) => {
          expectTypeOf(x).toEqualTypeOf<string>();
          return 0;
        },
      )
      .otherwise(() => 1);
  });
});
