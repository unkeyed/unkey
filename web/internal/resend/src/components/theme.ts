import type { CSSProperties } from "react";

export const colors = {
  light: {
    background: "#f9fafb",
    surface: "#ffffff",
    text: "#121317",
    muted: "#60606c",
    border: "#e3e4e9",
    button: "#121317",
    buttonText: "#ffffff",
  },
  dark: {
    background: "#040406",
    surface: "#121317",
    text: "#eeeff1",
    muted: "#abaebb",
    border: "#464853",
    button: "#eeeff1",
    buttonText: "#121317",
  },
};

const darkRules = {
  ".email-background": `background-color: ${colors.dark.background} !important;`,
  ".email-surface": `background-color: ${colors.dark.surface} !important;`,
  ".email-text": `color: ${colors.dark.text} !important;`,
  ".email-muted": `color: ${colors.dark.muted} !important;`,
  ".email-rule": `border-color: ${colors.dark.border} !important;`,
  ".email-button": `background-color: ${colors.dark.button} !important; color: ${colors.dark.buttonText} !important;`,
};

export const stylesheet = `
  :root { color-scheme: light dark; supported-color-schemes: light dark; }
  @media (prefers-color-scheme: dark) {
    ${Object.entries(darkRules)
      .map(([selector, rule]) => `${selector} { ${rule} }`)
      .join("\n")}
  }
  ${Object.entries(darkRules)
    .map(([selector, rule]) => `[data-ogsc] ${selector} { ${rule} }`)
    .join("\n")}
  @media only screen and (max-width: 600px) {
    .email-content { padding: 28px 20px !important; }
    .email-heading { font-size: 26px !important; line-height: 32px !important; }
  }
`;

export const styles = {
  body: {
    backgroundColor: colors.light.background,
    color: colors.light.text,
    fontFamily: "Arial, Helvetica, sans-serif",
    margin: "0",
    padding: "0",
    WebkitTextSizeAdjust: "100%",
  },
  container: {
    backgroundColor: colors.light.surface,
    width: "100%",
    maxWidth: "600px",
    margin: "0 auto",
    tableLayout: "fixed",
    overflowWrap: "anywhere",
    wordBreak: "break-word",
  },
  brand: {
    color: colors.light.text,
    fontSize: "24px",
    fontWeight: "700",
    letterSpacing: "-1px",
    textDecoration: "none",
  },
  heading: {
    color: colors.light.text,
    fontSize: "30px",
    lineHeight: "36px",
    fontWeight: "600",
    letterSpacing: "-0.6px",
    margin: "0 0 20px",
  },
  text: {
    color: colors.light.text,
    fontSize: "16px",
    lineHeight: "26px",
    margin: "0 0 16px",
  },
  muted: {
    color: colors.light.muted,
    fontSize: "13px",
    lineHeight: "20px",
    margin: "0 0 12px",
  },
  link: {
    color: colors.light.text,
    textDecoration: "underline",
    textUnderlineOffset: "3px",
  },
  rule: {
    border: "0",
    borderTop: `1px solid ${colors.light.border}`,
    margin: "28px 0",
  },
  button: {
    backgroundColor: colors.light.button,
    color: colors.light.buttonText,
    borderRadius: "6px",
    padding: "14px 24px",
    fontSize: "16px",
    fontWeight: "600",
    textAlign: "center",
    boxSizing: "border-box",
    textDecoration: "none",
  },
  sectionHeading: {
    color: colors.light.text,
    fontSize: "16px",
    lineHeight: "24px",
    fontWeight: "600",
    margin: "24px 0 12px",
  },
  list: {
    color: colors.light.text,
    fontSize: "16px",
    lineHeight: "26px",
    margin: "0 0 16px",
    paddingLeft: "20px",
  },
} satisfies Record<string, CSSProperties>;
