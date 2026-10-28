import { Button } from "@react-email/button";
import { Head } from "@react-email/head";
import { Heading } from "@react-email/heading";
import { Html } from "@react-email/html";
import { Img } from "@react-email/img";
import { Link } from "@react-email/link";
import { Preview } from "@react-email/preview";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
import { Children, Fragment, type ReactElement } from "react";
import { Signature } from "./signature";
import { styles, stylesheet } from "./theme";

type EmailLink = { label: string; href: string };

type EmailSection =
  | { type: "text"; text: string }
  | { type: "list"; title: string; items: readonly string[] }
  | { type: "links"; links: readonly EmailLink[] };

interface LayoutProps {
  title: string;
  preview: string;
  body: readonly (string | ReactElement)[];
  action: EmailLink;
  details?: readonly { label: string; value: string }[];
  sections?: readonly EmailSection[];
  socialLinks?: readonly EmailLink[];
  signedBy?: string;
  reason?: string;
  year?: string;
}

export function Layout({
  title,
  preview,
  body,
  action,
  details = [],
  sections = [],
  socialLinks = [],
  signedBy,
  reason,
  year,
}: LayoutProps) {
  return (
    <Html lang="en" dir="ltr">
      <Head>
        <title>{title}</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <meta name="color-scheme" content="light dark" />
        <meta name="supported-color-schemes" content="light dark" />
        <style>{stylesheet}</style>
      </Head>
      <body className="email-background" style={styles.body}>
        <Preview lang="en" dir="ltr" data-skip-in-text="true">
          {preview}
        </Preview>
        <table
          role="presentation"
          align="center"
          width="100%"
          cellPadding="0"
          cellSpacing="0"
          border={0}
          className="email-background"
          style={styles.background}
        >
          <tbody>
            <tr>
              <td>
                <table
                  role="presentation"
                  align="center"
                  width="560"
                  cellPadding="0"
                  cellSpacing="0"
                  border={0}
                  style={styles.container}
                >
                  <tbody>
                    <tr>
                      <td className="email-gutter" style={styles.gutter}>
                        <BrandHeader />
                        <Heading as="h1" className="email-text" style={styles.heading}>
                          {title}
                        </Heading>
                        {Children.map(body, (paragraph) => (
                          <Text className="email-copy" style={styles.text}>
                            {paragraph}
                          </Text>
                        ))}
                        {details.length > 0 && <Details details={details} />}
                        <Section style={{ paddingTop: "8px" }}>
                          <Button href={action.href} className="email-button" style={styles.button}>
                            {action.label}
                          </Button>
                        </Section>
                        {sections.length > 0 && <ContentSections sections={sections} />}
                        {signedBy && <Signature signedBy={signedBy} />}
                        <Section className="email-rule" style={styles.footer}>
                          <Text className="email-muted" style={styles.muted}>
                            Need help? Contact{" "}
                            <Link
                              href="mailto:support@unkey.com"
                              className="email-muted"
                              style={{ ...styles.link, ...styles.muted }}
                            >
                              support@unkey.com
                            </Link>
                            .
                          </Text>
                          {reason && (
                            <Text className="email-muted" style={styles.muted}>
                              {reason}
                            </Text>
                          )}
                          {year && (
                            <Text className="email-muted" style={styles.muted}>
                              © {year} Unkey
                            </Text>
                          )}
                          {socialLinks.length > 0 && (
                            <Text className="email-muted" style={styles.muted}>
                              {socialLinks.map((link, index) => (
                                <Fragment key={link.href}>
                                  {index > 0 && " · "}
                                  <Link
                                    href={link.href}
                                    className="email-muted"
                                    style={{ ...styles.link, ...styles.muted }}
                                  >
                                    {link.label}
                                  </Link>
                                </Fragment>
                              ))}
                            </Text>
                          )}
                        </Section>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </td>
            </tr>
          </tbody>
        </table>
      </body>
    </Html>
  );
}

function BrandHeader() {
  return (
    <table role="presentation" width="100%" cellPadding="0" cellSpacing="0" style={styles.header}>
      <tbody>
        <tr>
          <td>
            <Link href="https://www.unkey.com" aria-label="Unkey" data-skip-in-text="true">
              <Img
                src="https://app.unkey.com/images/emails/wordmark-light.png"
                alt="Unkey"
                width="83"
                height="28"
                className="email-logo-light email-text"
                style={styles.logo}
              />
              <Img
                src="https://app.unkey.com/images/emails/wordmark-dark.png"
                alt="Unkey"
                width="83"
                height="28"
                className="email-logo-dark email-text"
                style={{ ...styles.logo, display: "none", maxHeight: "0", overflow: "hidden" }}
              />
            </Link>
          </td>
        </tr>
      </tbody>
    </table>
  );
}

function Details({ details }: { details: NonNullable<LayoutProps["details"]> }) {
  return (
    <Section className="email-rule" style={styles.details}>
      <table width="100%" cellPadding="0" cellSpacing="0" style={{ tableLayout: "fixed" }}>
        <tbody>
          {details.map(({ label, value }) => (
            <tr key={label}>
              <th scope="row" className="email-muted" style={styles.detailLabel}>
                <p style={{ margin: "0" }}>{label}</p>
              </th>
              <td className="email-text" style={styles.detailValue}>
                <p style={{ margin: "0" }}>{value}</p>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Section>
  );
}

function ContentSections({ sections }: { sections: readonly EmailSection[] }) {
  return (
    <Section style={styles.sections}>
      {sections.map((section) => {
        switch (section.type) {
          case "text":
            return (
              <Text key={section.text} className="email-copy" style={styles.text}>
                {section.text}
              </Text>
            );
          case "list":
            return (
              <Section key={section.title}>
                <Heading as="h2" className="email-text" style={styles.sectionHeading}>
                  {section.title}
                </Heading>
                <ul className="email-copy" style={styles.list}>
                  {section.items.map((item) => (
                    <li key={item} style={{ paddingBottom: "4px" }}>
                      {item}
                    </li>
                  ))}
                </ul>
              </Section>
            );
          case "links":
            return (
              <Section key={section.links.map((link) => link.href).join(" ")} style={styles.links}>
                {section.links.map((link) => (
                  <Text key={link.href} style={styles.linkRow}>
                    <Link href={link.href} className="email-text" style={styles.link}>
                      {link.label}
                    </Link>
                  </Text>
                ))}
              </Section>
            );
        }
      })}
    </Section>
  );
}
