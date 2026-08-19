"use client";

import type { Route } from "next";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";

export function useConsumedSearchParam<T>(
  name: string,
  parse: (value: string | null) => T | null,
  cleanHref: Route,
): T | null {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [value] = useState(() => parse(searchParams?.get(name) ?? null));
  const consumed = useRef(false);

  useEffect(() => {
    if (value !== null && !consumed.current) {
      consumed.current = true;
      router.replace(cleanHref);
    }
  }, [value, router, cleanHref]);

  return value;
}
