module.exports = {
  plugins: {
    '@tailwindcss/postcss': {},
    "postcss-focus-visible": {
      replaceWith: "[data-focus-visible-added]",
    },
  },
};
