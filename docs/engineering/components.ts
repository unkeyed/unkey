import { defineComponents } from "blume";
import Callout from "blume/components/content/Callout.astro";
import AccordionGroup from "./components/AccordionGroup.astro";
import AccordionItem from "./components/AccordionItem.astro";
import Field from "./components/Field.astro";
import Warning from "./components/Warning.astro";

export default defineComponents({
  mdx: {
    Accordion: AccordionItem,
    AccordionGroup,
    ConfigToml: "./snippets/config-toml.mdx",
    Info: Callout,
    Note: Callout,
    RequestField: Field,
    ResponseField: Field,
    Warning,
  },
});
