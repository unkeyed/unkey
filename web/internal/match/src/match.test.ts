import { describe, expect, it } from "vitest";
import { NonExhaustiveError, P, match } from "./index";

describe("match", () => {
  describe(".with()", () => {
    it("matches a string literal", () => {
      const result = match("a" as "a" | "b")
        .with("a", () => 1)
        .with("b", () => 2)
        .exhaustive();
      expect(result).toBe(1);
    });

    it("matches a number literal", () => {
      const result = match(42 as number)
        .with(42, () => "found")
        .otherwise(() => "not found");
      expect(result).toBe("found");
    });

    it("matches an object pattern on a discriminated union", () => {
      type Shape = { kind: "circle"; r: number } | { kind: "rect"; w: number; h: number };
      const shape = { kind: "circle", r: 5 } as Shape;

      const area = match(shape)
        .with({ kind: "circle" }, (s) => Math.PI * s.r ** 2)
        .with({ kind: "rect" }, (s) => s.w * s.h)
        .exhaustive();

      expect(area).toBeCloseTo(Math.PI * 25);
    });

    it("falls through unmatched .with() to .otherwise()", () => {
      const result = match("c" as string)
        .with("a", () => 1)
        .with("b", () => 2)
        .otherwise(() => -1);
      expect(result).toBe(-1);
    });

    it("short-circuits after first match", () => {
      let callCount = 0;
      match("a" as "a" | "b")
        .with("a", () => {
          callCount++;
          return 1;
        })
        .with("b", () => {
          callCount++;
          return 2;
        })
        .otherwise(() => {
          callCount++;
          return 3;
        });
      expect(callCount).toBe(1);
    });
  });

  describe(".with() + guard", () => {
    it("matches when guard returns true", () => {
      const result = match(5 as number)
        .with(
          5,
          (n) => n > 0,
          () => "positive five",
        )
        .otherwise(() => "other");
      expect(result).toBe("positive five");
    });

    it("skips when guard returns false", () => {
      const result = match(5 as number)
        .with(
          5,
          (n) => n < 0,
          () => "negative five",
        )
        .otherwise(() => "other");
      expect(result).toBe("other");
    });
  });

  describe(".with() multi-pattern", () => {
    it("matches either pattern (OR semantics)", () => {
      const result = match("b" as "a" | "b" | "c")
        .with("a", "b", () => "first two")
        .with("c", () => "third")
        .exhaustive();
      expect(result).toBe("first two");
    });
  });

  describe(".when()", () => {
    it("matches using a predicate", () => {
      const result = match(10 as number)
        .when(
          (n) => n > 5,
          () => "big",
        )
        .otherwise(() => "small");
      expect(result).toBe("big");
    });

    it("skips when predicate returns false", () => {
      const result = match(3 as number)
        .when(
          (n) => n > 5,
          () => "big",
        )
        .otherwise(() => "small");
      expect(result).toBe("small");
    });
  });

  describe(".exhaustive()", () => {
    it("throws NonExhaustiveError when no pattern matched", () => {
      expect(() =>
        match("x" as string)
          .with("a", () => 1)
          // @ts-expect-error - intentionally testing runtime error for non-exhaustive match
          .exhaustive(),
      ).toThrow(NonExhaustiveError);
    });

    it("includes input in error message as JSON", () => {
      try {
        match({ foo: "bar" } as object)
          // @ts-expect-error - intentionally testing runtime error for non-exhaustive match
          .exhaustive();
      } catch (e) {
        expect(e).toBeInstanceOf(NonExhaustiveError);
        expect((e as NonExhaustiveError).message).toContain('{"foo":"bar"}');
        expect((e as NonExhaustiveError).input).toEqual({ foo: "bar" });
      }
    });
  });

  describe(".run()", () => {
    it("delegates to .exhaustive()", () => {
      const result = match("a" as "a" | "b")
        .with("a", () => 1)
        .with("b", () => 2)
        .run();
      expect(result).toBe(1);
    });
  });

  describe(".otherwise()", () => {
    it("returns matched value when matched", () => {
      const result = match("a")
        .with("a", () => 1)
        .otherwise(() => -1);
      expect(result).toBe(1);
    });

    it("calls handler with input when unmatched", () => {
      const result = match("z" as "a" | "z")
        .with("a", () => 1)
        .otherwise((v) => v);
      expect(result).toBe("z");
    });
  });

  describe(".returnType()", () => {
    it("is a runtime no-op", () => {
      const result = match("a" as "a" | "b")
        .returnType<number>()
        .with("a", () => 1)
        .with("b", () => 2)
        .exhaustive();
      expect(result).toBe(1);
    });
  });
});

describe("P", () => {
  describe("P.nullish", () => {
    it("matches null", () => {
      const result = match(null as string | null)
        .with(P.nullish, () => "empty")
        .otherwise(() => "has value");
      expect(result).toBe("empty");
    });

    it("matches undefined", () => {
      const result = match(undefined as string | undefined)
        .with(P.nullish, () => "empty")
        .otherwise(() => "has value");
      expect(result).toBe("empty");
    });

    it("does not match a string", () => {
      const result = match("hello" as string | null)
        .with(P.nullish, () => "empty")
        .otherwise(() => "has value");
      expect(result).toBe("has value");
    });
  });

  describe("P.nonNullable", () => {
    it("matches objects", () => {
      const result = match({ x: 1 } as { x: number } | null)
        .with(P.nonNullable, (v) => v.x)
        .otherwise(() => -1);
      expect(result).toBe(1);
    });

    it("does not match null", () => {
      const result = match(null as string | null)
        .with(P.nonNullable, () => "value")
        .otherwise(() => "empty");
      expect(result).toBe("empty");
    });
  });

  describe("P.string", () => {
    it("matches strings", () => {
      const result = match("hello" as string | number)
        .with(P.string, () => "is string")
        .otherwise(() => "not string");
      expect(result).toBe("is string");
    });

    it("does not match numbers", () => {
      const result = match(42 as string | number)
        .with(P.string, () => "is string")
        .otherwise(() => "not string");
      expect(result).toBe("not string");
    });
  });

  describe("P.number", () => {
    it("matches numbers", () => {
      const result = match(42 as string | number)
        .with(P.number, () => "is number")
        .otherwise(() => "not number");
      expect(result).toBe("is number");
    });
  });

  describe("P.boolean", () => {
    it("matches booleans", () => {
      const result = match(true as boolean | string)
        .with(P.boolean, () => "is bool")
        .otherwise(() => "not bool");
      expect(result).toBe("is bool");
    });
  });

  describe("P._", () => {
    it("matches anything", () => {
      const result = match("whatever" as string)
        .with(P._, () => "caught")
        .otherwise(() => "unreachable");
      expect(result).toBe("caught");
    });
  });

  describe("P.when()", () => {
    it("matches when predicate returns true", () => {
      const result = match(10 as number)
        .with(
          P.when((n: number) => n > 5),
          () => "big",
        )
        .otherwise(() => "small");
      expect(result).toBe("big");
    });
  });

  describe("P.nullish + guard", () => {
    it("matches null with passing guard", () => {
      const result = match(null as string | null)
        .with(
          P.nullish,
          () => true,
          () => "null + guard passed",
        )
        .otherwise(() => "fallback");
      expect(result).toBe("null + guard passed");
    });

    it("skips null when guard fails", () => {
      const result = match(null as string | null)
        .with(
          P.nullish,
          () => false,
          () => "null + guard passed",
        )
        .otherwise(() => "fallback");
      expect(result).toBe("fallback");
    });
  });

  describe("object patterns with P helpers", () => {
    it("matches object with P.string field", () => {
      type Data = { error: string | null; ok: boolean };
      const data: Data = { error: "bad", ok: false };

      const result = match(data)
        .with({ error: P.string }, () => "has error")
        .otherwise(() => "no error");
      expect(result).toBe("has error");
    });

    it("does not match when field is null", () => {
      type Data = { error: string | null; ok: boolean };
      const data: Data = { error: null, ok: true };

      const result = match(data)
        .with({ error: P.string }, () => "has error")
        .otherwise(() => "no error");
      expect(result).toBe("no error");
    });
  });
});

describe("Object.is semantics", () => {
  it("NaN matches NaN", () => {
    const result = match(Number.NaN)
      .with(Number.NaN, () => "nan")
      .otherwise(() => "other");
    expect(result).toBe("nan");
  });

  it("0 does not match -0", () => {
    const result = match(-0 as number)
      .with(0, () => "zero")
      .otherwise(() => "neg zero");
    expect(result).toBe("neg zero");
  });
});
