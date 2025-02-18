import {
  architectureSource,
  companySource,
  componentSource,
  contributingSource,
  rfcSource,
  source,
} from "@/app/source";
import { createSearchAPI } from "fumadocs-core/search/server";

const indexes = [
  source,
  rfcSource,
  companySource,
  componentSource,
  architectureSource,
  contributingSource,
].flatMap((src) =>
  src.getPages().map((page) => ({
    title: page.data.title,
    description: page.data.description,
    structuredData: page.data.structuredData,
    id: page.url,
    url: page.url,
  })),
);

export const { GET } = createSearchAPI("advanced", {
  indexes,
});
