//! Hand-rolled encoding helpers so we don't need base64/hex/url/html-escape
//! crates: the amounts of code involved are small and fully testable.

const B64_STD: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

/// Decodes base64, accepting both the standard and URL-safe alphabets, with
/// or without padding.
#[cfg_attr(not(test), allow(dead_code))]
pub fn base64_decode(input: &str) -> Result<Vec<u8>, String> {
    let mut out = Vec::with_capacity(input.len() / 4 * 3 + 3);
    let mut acc: u32 = 0;
    let mut nbits = 0u8;
    for c in input.bytes() {
        let v = match c {
            b'A'..=b'Z' => c - b'A',
            b'a'..=b'z' => c - b'a' + 26,
            b'0'..=b'9' => c - b'0' + 52,
            b'+' | b'-' => 62,
            b'/' | b'_' => 63,
            b'=' | b'\n' | b'\r' => continue,
            _ => return Err(format!("invalid base64 character {:?}", c as char)),
        };
        acc = (acc << 6) | v as u32;
        nbits += 6;
        if nbits >= 8 {
            nbits -= 8;
            out.push((acc >> nbits) as u8);
        }
    }
    Ok(out)
}

/// Encodes bytes as standard base64 with padding.
pub fn base64_encode(data: &[u8]) -> String {
    let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
    for chunk in data.chunks(3) {
        let b = [
            chunk[0],
            chunk.get(1).copied().unwrap_or(0),
            chunk.get(2).copied().unwrap_or(0),
        ];
        let n = (u32::from(b[0]) << 16) | (u32::from(b[1]) << 8) | u32::from(b[2]);
        out.push(B64_STD[(n >> 18) as usize & 63] as char);
        out.push(B64_STD[(n >> 12) as usize & 63] as char);
        out.push(if chunk.len() > 1 {
            B64_STD[(n >> 6) as usize & 63] as char
        } else {
            '='
        });
        out.push(if chunk.len() > 2 {
            B64_STD[n as usize & 63] as char
        } else {
            '='
        });
    }
    out
}

/// Percent-decodes a URL component. '+' decodes to space (query semantics).
pub fn percent_decode(input: &str) -> String {
    percent_decode_inner(input, true)
}

fn percent_decode_inner(input: &str, plus_as_space: bool) -> String {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        match bytes[i] {
            b'%' if i + 2 < bytes.len() + 1 && i + 2 < bytes.len() + 1 => {
                let hi = bytes.get(i + 1).and_then(|c| (*c as char).to_digit(16));
                let lo = bytes.get(i + 2).and_then(|c| (*c as char).to_digit(16));
                match (hi, lo) {
                    (Some(h), Some(l)) => {
                        out.push((h * 16 + l) as u8);
                        i += 3;
                    }
                    _ => {
                        out.push(b'%');
                        i += 1;
                    }
                }
            }
            b'+' => {
                out.push(if plus_as_space { b' ' } else { b'+' });
                i += 1;
            }
            b => {
                out.push(b);
                i += 1;
            }
        }
    }
    String::from_utf8_lossy(&out).into_owned()
}

/// Parses an application/x-www-form-urlencoded query string into pairs.
/// A bare key (no '=') yields an empty value, matching Go's url.ParseQuery.
pub fn parse_query(raw: &str) -> Vec<(String, String)> {
    raw.split('&')
        .filter(|s| !s.is_empty())
        .map(|pair| match pair.split_once('=') {
            Some((k, v)) => (percent_decode(k), percent_decode(v)),
            None => (percent_decode(pair), String::new()),
        })
        .collect()
}

/// Escapes text for safe interpolation into HTML.
pub fn html_escape(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for c in s.chars() {
        match c {
            '&' => out.push_str("&amp;"),
            '<' => out.push_str("&lt;"),
            '>' => out.push_str("&gt;"),
            '"' => out.push_str("&quot;"),
            '\'' => out.push_str("&#39;"),
            _ => out.push(c),
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn base64_decode_std_and_urlsafe() {
        assert_eq!(base64_decode("aGVsbG8=").unwrap(), b"hello");
        assert_eq!(base64_decode("aGVsbG8").unwrap(), b"hello");
        // URL-safe alphabet: 0xfb 0xef -> "--8" std would be "++8"
        assert_eq!(base64_decode("--8=").unwrap(), vec![0xfb, 0xef]);
        assert_eq!(base64_decode("++8=").unwrap(), vec![0xfb, 0xef]);
        assert!(base64_decode("!!!").is_err());
    }

    #[test]
    fn base64_encode_roundtrip() {
        for data in [&b""[..], b"f", b"fo", b"foo", b"foob", b"fooba", b"foobar"] {
            assert_eq!(base64_decode(&base64_encode(data)).unwrap(), data);
        }
        assert_eq!(base64_encode(b"foobar"), "Zm9vYmFy");
        assert_eq!(base64_encode(b"foob"), "Zm9vYg==");
    }

    #[test]
    fn percent_decoding() {
        assert_eq!(percent_decode("a%20b+c"), "a b c");
        assert_eq!(percent_decode("100%"), "100%");
        assert_eq!(percent_decode("%E2%82%AC"), "€");
    }

    #[test]
    fn query_parsing() {
        let q = parse_query("tag=a&tag=b&debug&x=1%202");
        assert_eq!(
            q,
            vec![
                ("tag".to_string(), "a".to_string()),
                ("tag".to_string(), "b".to_string()),
                ("debug".to_string(), "".to_string()),
                ("x".to_string(), "1 2".to_string()),
            ]
        );
    }

    #[test]
    fn html_escaping() {
        assert_eq!(
            html_escape(r#"<script>"a"&'b'</script>"#),
            "&lt;script&gt;&quot;a&quot;&amp;&#39;b&#39;&lt;/script&gt;"
        );
    }
}
