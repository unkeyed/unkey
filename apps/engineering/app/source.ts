import { docs, meta, rfcs } from "@/.source";
import { loader } from "fumadocs-core/source";
import { createMDXSource } from "fumadocs-mdx";

export const source = loader({
  baseUrl: "/docs",
  source: createMDXSource(docs, meta),
});

export const rfcSource = loader({
  baseUrl: "/rfcs",
  source: createMDXSource(rfcs, []),
});
