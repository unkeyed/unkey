import { z } from "zod";
import { isBrowser } from "./utils";
import { usecases } from "@/app/templates/data";

/**
 * Form values for the templates page.
 */
export const schema = z.object({
  search: z.string().optional(),
  frameworks: z.array(z.string()),
  languages: z.array(z.string()),
  usecases: z.array(z.string()),
});

/**
 * Infer the type from the schema.
 */
export type TemplatesFormValues = z.infer<typeof schema>;

/**
 * Get default form values from URL query params.
 */
export const getDefaulTemplatesFormValues = () => {
  if (!isBrowser) {
    return {
      search: undefined,
      frameworks: [],
      languages: [],
      usecases: []
    };
  }

  const searchParams = new URLSearchParams(window.location.search);

  const search = searchParams.get("search");
  const frameworks = searchParams.getAll("framework");
  const languages = searchParams.getAll("language");
  const usecases = searchParams.getAll("usecase");

  return {
    search: search || undefined,
    frameworks: frameworks.length > 0 ? frameworks : [],
    languages: languages.length > 0 ? languages : [],
    usecases: usecases.length > 0 ? usecases : [],
  };
};

/**
 * Update URL query params from form values.
 */
export const updateUrl = (values: TemplatesFormValues) => {
  const searchParams = new URLSearchParams(window.location.search);

  if (values.search) {
    searchParams.set("search", values.search);
  } else {
    searchParams.delete("search");
  }

  if(values.usecases) {
    searchParams.delete("usecase");
    values.usecases.forEach((usecase) => {
      searchParams.append("usecase", usecase);
    });
  } else {
    searchParams.delete("usecase");
  }

  if (values.frameworks) {
    searchParams.delete("framework");
    values.frameworks.forEach((framework) => {
      searchParams.append("framework", framework);
    });
  } else {
    searchParams.delete("framework");
  }

  if (values.languages) {
    searchParams.delete("language");
    values.languages.forEach((language) => {
      searchParams.append("language", language);
    });
  } else {
    searchParams.delete("language");
  }

  createUrl(searchParams);
};

/**
 * Create a new URL with the given search params.
 */
const createUrl = (searchParams: URLSearchParams) => {
  const newUrl = searchParams.toString()
    ? `${window.location.pathname}?${searchParams.toString()}`
    : window.location.pathname;

  window.history.replaceState(
    {
      ...window.history.state,
      as: newUrl,
      url: newUrl,
    },
    "",
    newUrl,
  );
};
