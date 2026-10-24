import { Body } from "@react-email/body";
import { Button } from "@react-email/button";
import { Head } from "@react-email/head";
import { Heading } from "@react-email/heading";
import { Hr } from "@react-email/hr";
import { Html } from "@react-email/html";
import { Link } from "@react-email/link";
import { Preview } from "@react-email/preview";
import { Section } from "@react-email/section";
import { Text } from "@react-email/text";
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
  body: readonly string[];
  action: EmailLink;
  details?: readonly { label: string; value: string }[];
  sections?: readonly EmailSection[];
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
      <Body className="email-background" style={styles.body}>
        <Preview lang="en" dir="ltr" data-skip-in-text="true">
          {preview}
        </Preview>
        <table
          role="presentation"
          align="center"
          width="600"
          cellPadding="0"
          cellSpacing="0"
          border={0}
          lang="en"
          dir="ltr"
          className="email-surface"
          style={styles.container}
        >
          <tbody>
            <tr>
              <td className="email-content" style={{ padding: "40px" }}>
                <Link href="https://www.unkey.com" className="email-text" style={styles.brand}>
                  Unkey
                </Link>
                <Hr className="email-rule" style={styles.rule} />
                <Heading as="h1" className="email-heading email-text" style={styles.heading}>
                  {title}
                </Heading>
                {body.map((paragraph) => (
                  <Text key={paragraph} className="email-text" style={styles.text}>
                    {paragraph}
                  </Text>
                ))}
                <Section style={{ padding: "8px 0 24px" }}>
                  <Button href={action.href} className="email-button" style={styles.button}>
                    {action.label}
                  </Button>
                </Section>
                {details.length > 0 && (
                  <Section>
                    {details.map(({ label, value }) => (
                      <Text key={label} className="email-text" style={styles.text}>
                        <span className="email-muted" style={styles.muted}>
                          {label}
                        </span>
                        <br />
                        {value}
                      </Text>
                    ))}
                  </Section>
                )}
                {sections.map((section) => {
                  switch (section.type) {
                    case "text":
                      return (
                        <Text key={section.text} className="email-text" style={styles.text}>
                          {section.text}
                        </Text>
                      );
                    case "list":
                      return (
                        <Section key={section.title}>
                          <Heading as="h2" className="email-text" style={styles.sectionHeading}>
                            {section.title}
                          </Heading>
                          <ul className="email-text" style={styles.list}>
                            {section.items.map((item) => (
                              <li key={item} style={{ paddingBottom: "8px" }}>
                                {item}
                              </li>
                            ))}
                          </ul>
                        </Section>
                      );
                    case "links":
                      return (
                        <Section key={section.links.map((link) => link.href).join(" ")}>
                          {section.links.map((link) => (
                            <Text key={link.href} style={styles.text}>
                              <Link href={link.href} className="email-text" style={styles.link}>
                                {link.label}
                              </Link>
                            </Text>
                          ))}
                        </Section>
                      );
                  }
                })}
                {signedBy && <Signature signedBy={signedBy} />}
                <Hr className="email-rule" style={styles.rule} />
                <Text className="email-muted" style={styles.muted}>
                  Need help? Contact{" "}
                  <Link href="mailto:support@unkey.com" className="email-text" style={styles.link}>
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
              </td>
            </tr>
          </tbody>
        </table>
      </Body>
    </Html>
  );
}
