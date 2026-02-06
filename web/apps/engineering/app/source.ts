import { components, docs, meta, packageDocs, packageMeta } from "@/.source";
import { loader } from "fumadocs-core/source";
import { createMDXSource } from "fumadocs-mdx";

export const source = loader({
  baseUrl: "/docs",
  source: createMDXSource(docs, meta),
});

export const componentSource = loader({
  baseUrl: "/design",
  source: createMDXSource(components, []),
});

export const packageSource = loader({
  baseUrl: "/packages",
  source: createMDXSource(packageDocs, packageMeta),
});
