//! Minimal synchronous HTTP/1.1 helpers for the off-hot-path clients
//! (vault, ctrl, ClickHouse) and the Prometheus listener. The request hot
//! path runs on hyper (see gateway.rs).

use std::io::{self, BufReader, Read, Write};
use std::net::TcpStream;
use std::time::Duration;

/// Maximum bytes captured from streaming request/response bodies for
/// analytics. Port of zen.MaxBodyCapture.
pub const MAX_BODY_CAPTURE: usize = 1 << 20; // 1 MiB

const MAX_HEAD_BYTES: usize = 64 * 1024;
const MAX_HEADERS: usize = 256;

/// Case-insensitive multimap preserving insertion order, like http.Header.
#[derive(Debug, Clone, Default)]
pub struct Headers {
    entries: Vec<(String, String)>,
}

impl Headers {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn get(&self, name: &str) -> Option<&str> {
        self.entries
            .iter()
            .find(|(k, _)| k.eq_ignore_ascii_case(name))
            .map(|(_, v)| v.as_str())
    }

    /// Replaces all values for `name` with a single value.
    pub fn set(&mut self, name: &str, value: &str) {
        self.remove(name);
        self.entries.push((name.to_string(), value.to_string()));
    }

    /// Appends a value without removing existing ones.
    pub fn add(&mut self, name: &str, value: &str) {
        self.entries.push((name.to_string(), value.to_string()));
    }

    pub fn remove(&mut self, name: &str) {
        self.entries.retain(|(k, _)| !k.eq_ignore_ascii_case(name));
    }

    pub fn iter(&self) -> impl Iterator<Item = (&str, &str)> {
        self.entries.iter().map(|(k, v)| (k.as_str(), v.as_str()))
    }

    pub fn len(&self) -> usize {
        self.entries.len()
    }
}

/// How a message body is framed on the wire.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BodyKind {
    Empty,
    ContentLength(u64),
    Chunked,
    /// Response bodies terminated by connection close (HTTP/1.0 style).
    UntilClose,
}

/// Parsed request head. The Prometheus listener is the only server-side
/// consumer; it reads path + keep_alive and serves bodyless GETs.
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct RequestHead {
    pub method: String,
    pub path: String,
    pub keep_alive: bool,
    pub body: BodyKind,
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct ResponseHead {
    pub status: u16,
    pub reason: String,
    pub body: BodyKind,
}

/// Stream abstraction over plain TCP and TLS connections.
pub trait Stream: Read + Write + Send {}

impl Stream for TcpStream {}
impl Stream for rustls::StreamOwned<rustls::ServerConnection, TcpStream> {}
impl Stream for rustls::StreamOwned<rustls::ClientConnection, TcpStream> {}

/// A buffered connection. Reads are buffered; writes go straight through.
pub struct Conn {
    reader: BufReader<Box<dyn Stream>>,
}

impl Conn {
    pub fn new(stream: Box<dyn Stream>) -> Self {
        Self {
            reader: BufReader::with_capacity(16 * 1024, stream),
        }
    }

    pub fn stream_mut(&mut self) -> &mut dyn Stream {
        self.reader.get_mut().as_mut()
    }

    pub fn write_all(&mut self, buf: &[u8]) -> io::Result<()> {
        self.reader.get_mut().write_all(buf)
    }

    pub fn flush(&mut self) -> io::Result<()> {
        self.reader.get_mut().flush()
    }

    fn read_line_crlf(&mut self, limit: usize) -> io::Result<String> {
        let mut line = Vec::with_capacity(128);
        loop {
            let mut byte = [0u8; 1];
            let n = self.reader.read(&mut byte)?;
            if n == 0 {
                if line.is_empty() {
                    return Err(io::Error::new(io::ErrorKind::UnexpectedEof, "eof"));
                }
                break;
            }
            if byte[0] == b'\n' {
                break;
            }
            line.push(byte[0]);
            if line.len() > limit {
                return Err(io::Error::new(io::ErrorKind::InvalidData, "line too long"));
            }
        }
        if line.last() == Some(&b'\r') {
            line.pop();
        }
        String::from_utf8(line).map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "not utf-8"))
    }

    fn read_headers(&mut self) -> io::Result<Headers> {
        let mut headers = Headers::new();
        let mut total = 0usize;
        loop {
            let line = self.read_line_crlf(MAX_HEAD_BYTES)?;
            if line.is_empty() {
                return Ok(headers);
            }
            total += line.len();
            if total > MAX_HEAD_BYTES || headers.len() >= MAX_HEADERS {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidData,
                    "headers too large",
                ));
            }
            let (name, value) = line
                .split_once(':')
                .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidData, "malformed header"))?;
            headers.add(name.trim(), value.trim());
        }
    }

    /// Reads a request head. Returns None on a clean EOF before any bytes
    /// (keep-alive connection closed by the client).
    pub fn read_request_head(&mut self) -> io::Result<Option<RequestHead>> {
        let line = match self.read_line_crlf(8 * 1024) {
            Ok(l) => l,
            Err(e) if e.kind() == io::ErrorKind::UnexpectedEof => return Ok(None),
            Err(e) => return Err(e),
        };
        let mut parts = line.split_ascii_whitespace();
        let method = parts
            .next()
            .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidData, "bad request line"))?
            .to_string();
        let raw_target = parts
            .next()
            .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidData, "bad request line"))?;
        let version = parts.next().unwrap_or("HTTP/1.1");
        let http11 = version.eq_ignore_ascii_case("HTTP/1.1");

        let headers = self.read_headers()?;
        let path = raw_target
            .split('?')
            .next()
            .unwrap_or(raw_target)
            .to_string();

        let body = if headers
            .get("Transfer-Encoding")
            .is_some_and(|te| te.to_ascii_lowercase().contains("chunked"))
        {
            BodyKind::Chunked
        } else {
            match headers
                .get("Content-Length")
                .and_then(|v| v.trim().parse::<u64>().ok())
            {
                Some(0) | None => BodyKind::Empty,
                Some(n) => BodyKind::ContentLength(n),
            }
        };
        let conn_header = headers.get("Connection").unwrap_or("").to_ascii_lowercase();
        let keep_alive = if http11 {
            !conn_header.contains("close")
        } else {
            conn_header.contains("keep-alive")
        };

        Ok(Some(RequestHead {
            method,
            path,
            keep_alive,
            body,
        }))
    }

    pub fn read_response_head(&mut self, req_method: &str) -> io::Result<ResponseHead> {
        let line = self.read_line_crlf(8 * 1024)?;
        let mut parts = line.splitn(3, ' ');
        let _version = parts.next().unwrap_or("");
        let status: u16 = parts
            .next()
            .unwrap_or("")
            .parse()
            .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "bad status line"))?;
        let reason = parts.next().unwrap_or("").to_string();
        let headers = self.read_headers()?;

        let body = if req_method.eq_ignore_ascii_case("HEAD")
            || status / 100 == 1
            || status == 204
            || status == 304
        {
            BodyKind::Empty
        } else if headers
            .get("Transfer-Encoding")
            .is_some_and(|te| te.to_ascii_lowercase().contains("chunked"))
        {
            BodyKind::Chunked
        } else {
            match headers
                .get("Content-Length")
                .and_then(|v| v.trim().parse::<u64>().ok())
            {
                Some(0) => BodyKind::Empty,
                Some(n) => BodyKind::ContentLength(n),
                None => BodyKind::UntilClose,
            }
        };

        Ok(ResponseHead {
            status,
            reason,
            body,
        })
    }
}

/// Reads a body from `src` according to `kind`, writing it to `dst`.
pub fn copy_body(src: &mut Conn, kind: BodyKind, dst: Option<&mut dyn Write>) -> io::Result<u64> {
    let mut written = 0u64;
    let mut dst = dst;
    let emit = |chunk: &[u8], dst: &mut Option<&mut dyn Write>| -> io::Result<()> {
        if let Some(d) = dst.as_deref_mut() {
            d.write_all(chunk)?;
        }
        Ok(())
    };

    let mut buf = [0u8; 16 * 1024];
    match kind {
        BodyKind::Empty => {}
        BodyKind::ContentLength(mut remaining) => {
            while remaining > 0 {
                let want = buf.len().min(remaining as usize);
                let n = src.reader.read(&mut buf[..want])?;
                if n == 0 {
                    return Err(io::Error::new(
                        io::ErrorKind::UnexpectedEof,
                        "body shorter than content-length",
                    ));
                }
                emit(&buf[..n], &mut dst)?;
                written += n as u64;
                remaining -= n as u64;
            }
        }
        BodyKind::Chunked => loop {
            let size_line = src.read_line_crlf(1024)?;
            let size_str = size_line.split(';').next().unwrap_or("").trim();
            let mut remaining = u64::from_str_radix(size_str, 16)
                .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "bad chunk size"))?;
            if remaining == 0 {
                // Trailers: consume until empty line.
                loop {
                    let l = src.read_line_crlf(MAX_HEAD_BYTES)?;
                    if l.is_empty() {
                        break;
                    }
                }
                break;
            }
            while remaining > 0 {
                let want = buf.len().min(remaining as usize);
                let n = src.reader.read(&mut buf[..want])?;
                if n == 0 {
                    return Err(io::Error::new(io::ErrorKind::UnexpectedEof, "eof in chunk"));
                }
                emit(&buf[..n], &mut dst)?;
                written += n as u64;
                remaining -= n as u64;
            }
            let crlf = src.read_line_crlf(16)?;
            if !crlf.is_empty() {
                return Err(io::Error::new(io::ErrorKind::InvalidData, "bad chunk end"));
            }
        },
        BodyKind::UntilClose => loop {
            let n = match src.reader.read(&mut buf) {
                Ok(n) => n,
                Err(e) if e.kind() == io::ErrorKind::UnexpectedEof => 0,
                Err(e) => return Err(e),
            };
            if n == 0 {
                break;
            }
            emit(&buf[..n], &mut dst)?;
            written += n as u64;
        },
    }
    Ok(written)
}

/// Serializes a request head as a single buffered write.
pub fn write_request_head(
    out: &mut dyn Write,
    method: &str,
    target: &str,
    headers: &Headers,
) -> io::Result<()> {
    let mut head = String::with_capacity(256);
    head.push_str(method);
    head.push(' ');
    head.push_str(target);
    head.push_str(" HTTP/1.1\r\n");
    for (k, v) in headers.iter() {
        head.push_str(k);
        head.push_str(": ");
        head.push_str(v);
        head.push_str("\r\n");
    }
    head.push_str("\r\n");
    out.write_all(head.as_bytes())
}

pub fn write_response_head(
    out: &mut dyn Write,
    status: u16,
    reason: &str,
    headers: &Headers,
) -> io::Result<()> {
    let reason = if reason.is_empty() {
        default_reason(status)
    } else {
        reason
    };
    let mut head = String::with_capacity(256);
    head.push_str("HTTP/1.1 ");
    head.push_str(&status.to_string());
    head.push(' ');
    head.push_str(reason);
    head.push_str("\r\n");
    for (k, v) in headers.iter() {
        head.push_str(k);
        head.push_str(": ");
        head.push_str(v);
        head.push_str("\r\n");
    }
    head.push_str("\r\n");
    out.write_all(head.as_bytes())
}

pub fn default_reason(status: u16) -> &'static str {
    match status {
        200 => "OK",
        404 => "Not Found",
        500 => "Internal Server Error",
        _ => "",
    }
}

/// Dials a TCP connection with a timeout, resolving the address first.
pub fn dial(addr: &str, timeout: Duration) -> io::Result<TcpStream> {
    use std::net::ToSocketAddrs;
    let addrs: Vec<std::net::SocketAddr> = addr.to_socket_addrs()?.collect();
    let first = addrs
        .first()
        .ok_or_else(|| io::Error::new(io::ErrorKind::NotFound, "no addresses resolved"))?;
    let stream = TcpStream::connect_timeout(first, timeout)?;
    stream.set_nodelay(true)?;
    Ok(stream)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn conn_from(data: &[u8]) -> Conn {
        struct MemStream {
            data: std::io::Cursor<Vec<u8>>,
            out: Vec<u8>,
        }
        impl Read for MemStream {
            fn read(&mut self, buf: &mut [u8]) -> io::Result<usize> {
                self.data.read(buf)
            }
        }
        impl Write for MemStream {
            fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
                self.out.write(buf)
            }
            fn flush(&mut self) -> io::Result<()> {
                Ok(())
            }
        }
        impl Stream for MemStream {}
        Conn::new(Box::new(MemStream {
            data: std::io::Cursor::new(data.to_vec()),
            out: Vec::new(),
        }))
    }

    #[test]
    fn parses_request_head() {
        let mut c = conn_from(b"GET /metrics?x=1 HTTP/1.1\r\nHost: example.com\r\n\r\n");
        let req = c.read_request_head().unwrap().unwrap();
        assert_eq!(req.method, "GET");
        assert_eq!(req.path, "/metrics");
        assert_eq!(req.body, BodyKind::Empty);
        assert!(req.keep_alive);
    }

    #[test]
    fn parses_response_head_and_body() {
        let mut c = conn_from(b"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello");
        let resp = c.read_response_head("GET").unwrap();
        assert_eq!(resp.status, 200);
        assert_eq!(resp.body, BodyKind::ContentLength(5));
        let mut out = Vec::new();
        let n = copy_body(&mut c, resp.body, Some(&mut out)).unwrap();
        assert_eq!(n, 5);
        assert_eq!(out, b"hello");
    }

    #[test]
    fn parses_chunked_body() {
        let mut c = conn_from(b"5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n");
        let mut out = Vec::new();
        let n = copy_body(&mut c, BodyKind::Chunked, Some(&mut out)).unwrap();
        assert_eq!(n, 11);
        assert_eq!(out, b"hello world");
    }

    #[test]
    fn head_and_204_have_no_body() {
        let mut c = conn_from(b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n");
        assert_eq!(c.read_response_head("HEAD").unwrap().body, BodyKind::Empty);
        let mut c = conn_from(b"HTTP/1.1 204 No Content\r\n\r\n");
        assert_eq!(c.read_response_head("GET").unwrap().body, BodyKind::Empty);
    }

    #[test]
    fn headers_are_case_insensitive_and_ordered() {
        let mut h = Headers::new();
        h.add("X-One", "1");
        h.add("x-one", "2");
        h.set("Content-Type", "text/plain");
        assert_eq!(h.get("X-ONE"), Some("1"));
        h.remove("X-One");
        assert_eq!(h.get("x-one"), None);
        assert_eq!(h.get("content-type"), Some("text/plain"));
    }

    #[test]
    fn writes_heads_as_single_buffer() {
        let mut out = Vec::new();
        let mut h = Headers::new();
        h.set("Host", "example.com");
        write_request_head(&mut out, "POST", "/x", &h).unwrap();
        assert_eq!(
            String::from_utf8(out).unwrap(),
            "POST /x HTTP/1.1\r\nHost: example.com\r\n\r\n"
        );
    }
}
