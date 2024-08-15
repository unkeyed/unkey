/*
    Adapted from the Prism One Dark Theme
    https://github.com/PrismJS/prism-themes/blob/master/themes/prism-one-dark.css
    Created by Marc Rousavy (@mrousavy) on 26.9.2023
*/
import type { PrismTheme } from "prism-react-renderer";

const darkTheme: PrismTheme = {
  plain: {
    backgroundColor: "#282A36",
    color: "#F8F8F2",
    textShadow: "0 1px rgba(0, 0, 0, 0.3)",
  },
  styles: [
    {
      types: ["comment", "prolog", "cdata"],
      style: {
        color: "#F8F8F2",
      },
    },
    {
      types: ["doctype", "punctuation", "entity"],
      style: {
        color: "#F8F8F2",
      },
    },
    {
      types: [
        "attr-name",
        "class-name",
        "maybe-class-name",
        "boolean",
        "constant",
        "number",
        "atrule",
      ],
      style: { color: "#F8F8F2" },
    },
    {
      types: ["keyword"],
      style: { color: "#9D72FF" },
    },
    {
      types: ["property", "tag", "symbol", "deleted", "important"],
      style: {
        color: "#FB3186",
      },
    },

    {
      types: ["selector", "string", "char", "builtin", "inserted", "regex", "attr-value"],
      style: {
        color: "#3CEEAE",
      },
    },
    {
      types: ["variable", "operator", "function"],
      style: {
        color: "#FB3186",
      },
    },
    {
      types: ["url"],
      style: {
        color: "#3CEEAE",
      },
    },
    {
      types: ["deleted"],
      style: {
        textDecorationLine: "line-through",
      },
    },
    {
      types: ["inserted"],
      style: {
        textDecorationLine: "underline",
      },
    },
    {
      types: ["italic"],
      style: {
        fontStyle: "italic",
      },
    },
    {
      types: ["important", "bold"],
      style: {
        fontWeight: "bold",
      },
    },
    {
      types: ["important"],
      style: {
        color: "#FB3186",
      },
    },
  ],
};

export default darkTheme;
