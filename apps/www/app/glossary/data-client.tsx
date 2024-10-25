import { FileJson } from "lucide-react";
import { categories } from "./data";
// note this is a separate client-file to include the icons, so that we can load the typescript .ts file into our content-collection config
export const categoriesWithIcons = [
  ...categories.map((c) => ({ ...c, icon: <FileJson /> })),
] as const;
