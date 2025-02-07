import { architecture, company, components, contributing, docs, meta, rfcs } from "@/.source";
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

export const companySource = loader({
  baseUrl: "/company",
  source: createMDXSource(company, []),
});

export const componentSource = loader({
  baseUrl: "/design",
  source: createMDXSource(components, []),
});

export const contributingSource = loader({
  baseUrl: "/contributing",
  source: createMDXSource(contributing, []),
});
export const architectureSource = loader({
  baseUrl: "/architecture",
  source: createMDXSource(architecture, []),
});
