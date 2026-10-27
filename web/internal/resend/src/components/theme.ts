import type { CSSProperties } from "react";

const accent = "#def135";

export const colors = {
  light: {
    background: "#ffffff",
    text: "#121317",
    copy: "#464853",
    muted: "#60606c",
    border: "#e3e4e9",
    button: accent,
    buttonText: "#121317",
  },
  dark: {
    background: "#040406",
    text: "#eeeff1",
    copy: "#c7c9d1",
    muted: "#9194a1",
    border: "#2e3038",
    button: accent,
    buttonText: "#121317",
  },
};

const fontFamily = "Inter, Helvetica, Arial, sans-serif";

const darkRules = {
  ".email-background": `background-color: ${colors.dark.background} !important;`,
  ".email-text": `color: ${colors.dark.text} !important;`,
  ".email-copy": `color: ${colors.dark.copy} !important;`,
  ".email-muted": `color: ${colors.dark.muted} !important;`,
  ".email-rule": `border-color: ${colors.dark.border} !important;`,
  ".email-button": `background-color: ${colors.dark.button} !important; color: ${colors.dark.buttonText} !important;`,
  ".email-logo-light":
    "display: none !important; max-height: 0 !important; overflow: hidden !important;",
  ".email-logo-dark":
    "display: block !important; max-height: none !important; overflow: visible !important;",
};

export const stylesheet = `
  :root { color-scheme: light dark; supported-color-schemes: light dark; }
  @font-face {
    font-family: Inter;
    font-style: normal;
    font-weight: 400;
    font-display: swap;
    mso-font-alt: Arial;
    src: url(https://fonts.gstatic.com/s/inter/v20/UcCO3FwrK3iLTeHuS_nVMrMxCp50SjIw2boKoduKmMEVuLyfMZg.ttf);
  }
  @font-face {
    font-family: Inter;
    font-style: normal;
    font-weight: 500;
    font-display: swap;
    mso-font-alt: Arial;
    src: url(https://fonts.gstatic.com/s/inter/v20/UcCO3FwrK3iLTeHuS_nVMrMxCp50SjIw2boKoduKmMEVuI6fMZg.ttf);
  }
  @font-face {
    font-family: Inter;
    font-style: normal;
    font-weight: 600;
    font-display: swap;
    mso-font-alt: Arial;
    src: url(https://fonts.gstatic.com/s/inter/v20/UcCO3FwrK3iLTeHuS_nVMrMxCp50SjIw2boKoduKmMEVuGKYMZg.ttf);
  }
  .email-logo-dark { mso-hide: all; }
  @media (prefers-color-scheme: dark) {
    ${Object.entries(darkRules)
      .map(([selector, rule]) => `${selector} { ${rule} }`)
      .join("\n")}
  }
  ${Object.entries(darkRules)
    .map(([selector, rule]) => `[data-ogsc] ${selector} { ${rule} }`)
    .join("\n")}
  @media only screen and (max-width: 480px) {
    .email-gutter { padding: 32px 24px !important; }
  }
`;

export const styles = {
  body: {
    backgroundColor: colors.light.background,
    color: colors.light.text,
    fontFamily,
    margin: "0",
    padding: "0",
    WebkitTextSizeAdjust: "100%",
  },
  background: {
    backgroundColor: colors.light.background,
  },
  container: {
    width: "100%",
    maxWidth: "560px",
    margin: "0 auto",
    tableLayout: "fixed",
    overflowWrap: "anywhere",
    wordBreak: "break-word",
    fontFamily,
  },
  gutter: {
    padding: "48px 24px",
  },
  header: {
    paddingBottom: "32px",
  },
  logo: {
    display: "block",
    color: colors.light.text,
    fontSize: "20px",
  },
  heading: {
    color: colors.light.text,
    fontSize: "24px",
    lineHeight: "32px",
    fontWeight: "500",
    letterSpacing: "-0.4px",
    margin: "0 0 20px",
  },
  text: {
    color: colors.light.copy,
    fontSize: "16px",
    lineHeight: "24px",
    margin: "0 0 16px",
  },
  emphasis: {
    color: colors.light.text,
    fontWeight: "600",
  },
  muted: {
    color: colors.light.muted,
    fontSize: "13px",
    lineHeight: "20px",
    margin: "0 0 8px",
  },
  link: {
    color: colors.light.text,
    textDecoration: "underline",
    textUnderlineOffset: "3px",
  },
  sections: {
    marginTop: "32px",
  },
  links: {
    margin: "0 0 16px",
  },
  linkRow: {
    fontSize: "16px",
    lineHeight: "24px",
    margin: "0",
    padding: "4px 0",
  },
  button: {
    backgroundColor: colors.light.button,
    color: colors.light.buttonText,
    borderRadius: "5px",
    padding: "14px 18px",
    fontSize: "14px",
    lineHeight: "20px",
    fontWeight: "500",
    textAlign: "center",
    boxSizing: "border-box",
    textDecoration: "none",
  },
  details: {
    borderTop: `1px solid ${colors.light.border}`,
    borderBottom: `1px solid ${colors.light.border}`,
    padding: "12px 0",
    margin: "24px 0",
  },
  detailLabel: {
    color: colors.light.muted,
    fontSize: "14px",
    lineHeight: "22px",
    fontWeight: "400",
    padding: "4px 12px 4px 0",
    textAlign: "left",
    verticalAlign: "top",
    width: "28%",
  },
  detailValue: {
    color: colors.light.text,
    fontSize: "14px",
    lineHeight: "22px",
    padding: "4px 0",
    verticalAlign: "top",
  },
  sectionHeading: {
    color: colors.light.text,
    fontSize: "16px",
    lineHeight: "24px",
    fontWeight: "500",
    margin: "0 0 8px",
  },
  list: {
    color: colors.light.copy,
    fontSize: "16px",
    lineHeight: "24px",
    margin: "0 0 16px",
    paddingLeft: "18px",
  },
  footer: {
    borderTop: `1px solid ${colors.light.border}`,
    marginTop: "24px",
    paddingTop: "20px",
  },
} satisfies Record<string, CSSProperties>;
