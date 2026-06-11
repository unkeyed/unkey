//! unkey-frontline: Rust port of svc/frontline (routing/proxy scope).
//!
//! Usage: unkey-frontline --config /etc/unkey/frontline.toml

mod cache;
mod certmanager;
mod clickhouse;
mod config;
mod connectrpc;
mod db;
mod encoding;
mod error;
mod errorpage;
mod gateway;
mod httpx;
mod logger;
mod metrics;
mod router;
mod run;
mod session;
mod tlsx;
mod uid;

fn main() {
    // rustls needs a process-wide crypto provider before any TLS use.
    rustls::crypto::ring::default_provider()
        .install_default()
        .expect("install rustls crypto provider");

    match std::env::var("UNKEY_LOG_LEVEL").as_deref() {
        Ok("debug") => logger::set_min_level(logger::Level::Debug),
        Ok("warn") => logger::set_min_level(logger::Level::Warn),
        Ok("error") => logger::set_min_level(logger::Level::Error),
        _ => logger::set_min_level(logger::Level::Info),
    }

    let mut config_path = String::new();
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--config" | "-c" => {
                config_path = args.next().unwrap_or_default();
            }
            "--help" | "-h" => {
                println!("Usage: unkey-frontline --config <path-to-frontline.toml>");
                return;
            }
            other if config_path.is_empty() && !other.starts_with('-') => {
                config_path = other.to_string();
            }
            other => {
                eprintln!("unknown argument: {other}");
                std::process::exit(2);
            }
        }
    }
    if config_path.is_empty() {
        eprintln!("Usage: unkey-frontline --config <path-to-frontline.toml>");
        std::process::exit(2);
    }

    let cfg = match config::Config::load(&config_path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("failed to load config: {e}");
            std::process::exit(1);
        }
    };

    if let Err(e) = run::run(cfg) {
        log_error!("frontline failed", "error" => e);
        std::process::exit(1);
    }
}
