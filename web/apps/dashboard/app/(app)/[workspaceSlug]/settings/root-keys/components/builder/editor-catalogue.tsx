"use client";

import { createContext, useContext } from "react";
import { CATALOGUES } from "./lib/catalogue";
import type { ResourceScope, ScopeCatalogue } from "./lib/catalogue.types";
import { type RootKeyTemplate, TEMPLATES } from "./lib/templates";

export type EditorCatalogue = {
  catalogues: Record<ResourceScope, ScopeCatalogue>;
  templates: readonly RootKeyTemplate[];
};

const EditorCatalogueContext = createContext<EditorCatalogue>({
  catalogues: CATALOGUES,
  templates: TEMPLATES,
});

export const EditorCatalogueProvider = EditorCatalogueContext.Provider;

export function useEditorCatalogue(): EditorCatalogue {
  return useContext(EditorCatalogueContext);
}
