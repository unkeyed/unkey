//! Port of internal/errorpage: renders the HTML error page. The template is
//! the same as error.go.tmpl; the Go text/template conditionals are replaced
//! by simple string assembly with HTML escaping.

use crate::encoding::html_escape;

pub struct Data<'a> {
    pub status_code: u16,
    pub title: &'a str,
    pub message: &'a str,
    pub error_code: &'a str,
    pub docs_url: &'a str,
    pub request_id: &'a str,
}

const STYLE: &str = r#"
        :root {
            --bg: #09090b;
            --fg: #fafafa;
            --muted: #a1a1aa;
            --dim: #71717a;
            --faint: #52525b;
            --border: #27272a;
            --surface: #18181b;
        }

        @media (prefers-color-scheme: light) {
            :root {
                --bg: #fafafa;
                --fg: #09090b;
                --muted: #71717a;
                --dim: #a1a1aa;
                --faint: #a1a1aa;
                --border: #e4e4e7;
                --surface: #f4f4f5;
            }
        }

        *, *::before, *::after {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: ui-monospace, "Cascadia Mono", "Segoe UI Mono", "Liberation Mono", Menlo, Monaco, Consolas, monospace;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 2rem;
            background: var(--bg);
            color: var(--fg);
            -webkit-font-smoothing: antialiased;
        }

        .container {
            max-width: 480px;
            width: 100%;
        }

        .header {
            display: flex;
            align-items: baseline;
            gap: 0.75rem;
        }

        .status {
            font-size: 3rem;
            font-weight: 700;
            line-height: 1;
            letter-spacing: -0.03em;
        }

        .title {
            font-size: 1rem;
            font-weight: 400;
            color: var(--muted);
        }

        .message {
            margin-top: 1.25rem;
            padding: 1rem 1.25rem;
            font-size: 0.8125rem;
            line-height: 1.7;
            color: var(--muted);
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: 6px;
        }

        .meta {
            margin-top: 1.25rem;
            display: flex;
            flex-direction: column;
            gap: 0.375rem;
            font-size: 0.75rem;
        }

        .meta-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.25rem 0;
        }

        .meta-label {
            color: var(--faint);
            text-transform: uppercase;
            font-size: 0.6875rem;
            letter-spacing: 0.05em;
        }

        .meta-value {
            color: var(--muted);
        }

        .meta-value a {
            color: var(--muted);
            text-decoration: underline;
            text-decoration-color: var(--faint);
            text-underline-offset: 3px;
            transition: color 0.15s, text-decoration-color 0.15s;
        }

        .meta-value a:hover {
            color: var(--fg);
            text-decoration-color: var(--fg);
        }

        .footer {
            margin-top: 2rem;
            padding-top: 1rem;
            border-top: 1px solid var(--border);
            font-size: 0.6875rem;
            color: var(--faint);
        }

        .footer a {
            color: var(--dim);
            text-decoration: none;
        }

        .footer a:hover {
            color: var(--fg);
        }
"#;

pub fn render(data: &Data<'_>) -> String {
    let mut meta_rows = String::new();
    if !data.request_id.is_empty() {
        meta_rows.push_str(&format!(
            r#"            <div class="meta-row">
                <span class="meta-label">Request ID</span>
                <span class="meta-value">{}</span>
            </div>
"#,
            html_escape(data.request_id)
        ));
    }
    if !data.error_code.is_empty() {
        let code_html = if !data.docs_url.is_empty() {
            format!(
                r#"<a href="{}" target="_blank">{}</a>"#,
                html_escape(data.docs_url),
                html_escape(data.error_code)
            )
        } else {
            html_escape(data.error_code)
        };
        meta_rows.push_str(&format!(
            r#"            <div class="meta-row">
                <span class="meta-label">Code</span>
                <span class="meta-value">{code_html}</span>
            </div>
"#
        ));
    }

    format!(
        r#"<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{status} {title}</title>
    <style>{STYLE}    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="status">{status}</div>
            <div class="title">{title}</div>
        </div>

        <div class="message">{message}</div>

        <div class="meta">
{meta_rows}        </div>

        <div class="footer">
            Need help? <a href="mailto:support@unkey.com">support@unkey.com</a>
        </div>
    </div>
</body>
</html>"#,
        status = data.status_code,
        title = html_escape(data.title),
        message = html_escape(data.message),
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn renders_all_fields_escaped() {
        let html = render(&Data {
            status_code: 502,
            title: "Bad Gateway",
            message: "<script>alert(1)</script>",
            error_code: "err:frontline:upstream:bad_gateway",
            docs_url: "https://unkey.com/docs/errors/err/frontline/upstream/bad_gateway",
            request_id: "req_123",
        });
        assert!(html.contains("502"));
        assert!(html.contains("Bad Gateway"));
        assert!(html.contains("&lt;script&gt;"));
        assert!(!html.contains("<script>alert"));
        assert!(html.contains("req_123"));
        assert!(html.contains("err:frontline:upstream:bad_gateway"));
        assert!(html.contains("https://unkey.com/docs/errors/"));
    }

    #[test]
    fn omits_empty_meta_rows() {
        let html = render(&Data {
            status_code: 404,
            title: "Not Found",
            message: "nope",
            error_code: "",
            docs_url: "",
            request_id: "",
        });
        assert!(!html.contains("Request ID"));
        assert!(!html.contains("meta-label\">Code"));
    }
}
