"use client";

import { faker } from "@faker-js/faker";

export function generateSemanticCacheDefaultName() {
  return `${faker.hacker.adjective()}-${faker.hacker.adjective()}-${
    faker.science.chemicalElement().name
  }-${faker.number.int({ min: 1000, max: 9999 })}`
    .replaceAll(/\s+/g, "-")
    .toLowerCase();
}
