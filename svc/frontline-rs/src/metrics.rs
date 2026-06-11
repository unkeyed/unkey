//! Hand-rolled Prometheus-compatible metrics: counters, gauges, and
//! histograms with label support and text exposition, replacing the
//! prometheus client crate. Mirrors every metric frontline emits in Go
//! (names, labels, and buckets are identical).

use std::collections::HashMap;
use std::sync::atomic::{AtomicI64, AtomicU64, Ordering};
use std::sync::{Arc, LazyLock, Mutex};

#[derive(Default)]
pub struct Counter {
    value: AtomicU64,
}

impl Counter {
    pub fn inc(&self) {
        self.value.fetch_add(1, Ordering::Relaxed);
    }
    pub fn get(&self) -> u64 {
        self.value.load(Ordering::Relaxed)
    }
}

#[derive(Default)]
pub struct Gauge {
    value: AtomicI64,
}

impl Gauge {
    pub fn inc(&self) {
        self.value.fetch_add(1, Ordering::Relaxed);
    }
    pub fn dec(&self) {
        self.value.fetch_sub(1, Ordering::Relaxed);
    }
    pub fn get(&self) -> i64 {
        self.value.load(Ordering::Relaxed)
    }
}

pub struct Histogram {
    buckets: Vec<f64>,
    counts: Vec<AtomicU64>,
    sum_bits: AtomicU64,
    count: AtomicU64,
}

impl Histogram {
    fn new(buckets: &[f64]) -> Self {
        Self {
            buckets: buckets.to_vec(),
            counts: buckets.iter().map(|_| AtomicU64::new(0)).collect(),
            sum_bits: AtomicU64::new(0),
            count: AtomicU64::new(0),
        }
    }

    pub fn observe(&self, v: f64) {
        for (i, b) in self.buckets.iter().enumerate() {
            if v <= *b {
                self.counts[i].fetch_add(1, Ordering::Relaxed);
            }
        }
        self.count.fetch_add(1, Ordering::Relaxed);
        // CAS loop to add to the f64 sum stored as bits.
        let mut old = self.sum_bits.load(Ordering::Relaxed);
        loop {
            let new = f64::to_bits(f64::from_bits(old) + v);
            match self.sum_bits.compare_exchange_weak(
                old,
                new,
                Ordering::Relaxed,
                Ordering::Relaxed,
            ) {
                Ok(_) => break,
                Err(actual) => old = actual,
            }
        }
    }

    fn sum(&self) -> f64 {
        f64::from_bits(self.sum_bits.load(Ordering::Relaxed))
    }
}

enum MetricKind {
    Counter,
    Gauge,
    Histogram(&'static [f64]),
}

enum Children {
    Counters(Mutex<HashMap<Vec<String>, Arc<Counter>>>),
    Gauges(Mutex<HashMap<Vec<String>, Arc<Gauge>>>),
    Histograms(Mutex<HashMap<Vec<String>, Arc<Histogram>>>),
}

/// A metric family: one name + help + label names, many children keyed by
/// label values. Unlabeled metrics are families with a single child under
/// the empty label vector.
pub struct Family {
    name: &'static str,
    help: &'static str,
    labels: &'static [&'static str],
    kind: MetricKind,
    children: Children,
}

impl Family {
    pub fn counter_with(&self, label_values: &[&str]) -> Arc<Counter> {
        let key: Vec<String> = label_values.iter().map(|s| s.to_string()).collect();
        match &self.children {
            Children::Counters(m) => m.lock().unwrap().entry(key).or_default().clone(),
            _ => unreachable!("not a counter family"),
        }
    }

    pub fn gauge_with(&self, label_values: &[&str]) -> Arc<Gauge> {
        let key: Vec<String> = label_values.iter().map(|s| s.to_string()).collect();
        match &self.children {
            Children::Gauges(m) => m.lock().unwrap().entry(key).or_default().clone(),
            _ => unreachable!("not a gauge family"),
        }
    }

    pub fn histogram_with(&self, label_values: &[&str]) -> Arc<Histogram> {
        let key: Vec<String> = label_values.iter().map(|s| s.to_string()).collect();
        match (&self.children, &self.kind) {
            (Children::Histograms(m), MetricKind::Histogram(buckets)) => m
                .lock()
                .unwrap()
                .entry(key)
                .or_insert_with(|| Arc::new(Histogram::new(buckets)))
                .clone(),
            _ => unreachable!("not a histogram family"),
        }
    }
}

pub struct Registry {
    families: Mutex<Vec<&'static Family>>,
}

pub static REGISTRY: LazyLock<Registry> = LazyLock::new(|| Registry {
    families: Mutex::new(Vec::new()),
});

impl Registry {
    fn register(&self, f: &'static Family) {
        self.families.lock().unwrap().push(f);
    }

    /// Renders all metrics in the Prometheus text exposition format.
    pub fn gather(&self) -> String {
        let mut out = String::with_capacity(4096);
        for f in self.families.lock().unwrap().iter() {
            let type_str = match f.kind {
                MetricKind::Counter => "counter",
                MetricKind::Gauge => "gauge",
                MetricKind::Histogram(_) => "histogram",
            };
            out.push_str(&format!("# HELP {} {}\n", f.name, f.help));
            out.push_str(&format!("# TYPE {} {}\n", f.name, type_str));
            match &f.children {
                Children::Counters(m) => {
                    for (lv, c) in m.lock().unwrap().iter() {
                        out.push_str(&format!(
                            "{}{} {}\n",
                            f.name,
                            render_labels(f.labels, lv),
                            c.get()
                        ));
                    }
                }
                Children::Gauges(m) => {
                    for (lv, g) in m.lock().unwrap().iter() {
                        out.push_str(&format!(
                            "{}{} {}\n",
                            f.name,
                            render_labels(f.labels, lv),
                            g.get()
                        ));
                    }
                }
                Children::Histograms(m) => {
                    for (lv, h) in m.lock().unwrap().iter() {
                        for (i, b) in h.buckets.iter().enumerate() {
                            out.push_str(&format!(
                                "{}_bucket{} {}\n",
                                f.name,
                                render_labels_with(f.labels, lv, Some(("le", &format_f64(*b)))),
                                h.counts[i].load(Ordering::Relaxed)
                            ));
                        }
                        out.push_str(&format!(
                            "{}_bucket{} {}\n",
                            f.name,
                            render_labels_with(f.labels, lv, Some(("le", "+Inf"))),
                            h.count.load(Ordering::Relaxed)
                        ));
                        out.push_str(&format!(
                            "{}_sum{} {}\n",
                            f.name,
                            render_labels(f.labels, lv),
                            format_f64(h.sum())
                        ));
                        out.push_str(&format!(
                            "{}_count{} {}\n",
                            f.name,
                            render_labels(f.labels, lv),
                            h.count.load(Ordering::Relaxed)
                        ));
                    }
                }
            }
        }
        out
    }
}

fn format_f64(v: f64) -> String {
    if v == v.trunc() && v.abs() < 1e15 {
        format!("{v:.0}")
    } else {
        format!("{v}")
    }
}

fn render_labels(names: &[&str], values: &[String]) -> String {
    render_labels_with(names, values, None)
}

fn render_labels_with(names: &[&str], values: &[String], extra: Option<(&str, &str)>) -> String {
    if names.is_empty() && extra.is_none() {
        return String::new();
    }
    let mut parts: Vec<String> = names
        .iter()
        .zip(values.iter())
        .map(|(n, v)| format!("{}=\"{}\"", n, v.replace('\\', "\\\\").replace('"', "\\\"")))
        .collect();
    if let Some((k, v)) = extra {
        parts.push(format!("{k}=\"{v}\""));
    }
    format!("{{{}}}", parts.join(","))
}

fn family(
    name: &'static str,
    help: &'static str,
    labels: &'static [&'static str],
    kind: MetricKind,
) -> Family {
    let children = match kind {
        MetricKind::Counter => Children::Counters(Mutex::new(HashMap::new())),
        MetricKind::Gauge => Children::Gauges(Mutex::new(HashMap::new())),
        MetricKind::Histogram(_) => Children::Histograms(Mutex::new(HashMap::new())),
    };
    Family {
        name,
        help,
        labels,
        kind,
        children,
    }
}

macro_rules! register_family {
    ($static_name:ident, $name:expr, $help:expr, $labels:expr, $kind:expr) => {
        pub static $static_name: LazyLock<&'static Family> = LazyLock::new(|| {
            let f: &'static Family = Box::leak(Box::new(family($name, $help, $labels, $kind)));
            REGISTRY.register(f);
            f
        });
    };
}

// ---------------------------------------------------------------------------
// Frontline metric families (identical names/labels/buckets to the Go service)
// ---------------------------------------------------------------------------

register_family!(
    REQUESTS_TOTAL,
    "unkey_frontline_requests_total",
    "Total requests by status class, attribution domain, and fault code (URN).",
    &["status_class", "fault_domain", "code"],
    MetricKind::Counter
);

register_family!(
    INFLIGHT_REQUESTS,
    "unkey_frontline_inflight_requests",
    "Requests currently being processed.",
    &[],
    MetricKind::Gauge
);

register_family!(
    HTTPS_REDIRECTS_TOTAL,
    "unkey_frontline_https_redirects_total",
    "Total number of plain-HTTP requests upgraded to HTTPS via 308.",
    &[],
    MetricKind::Counter
);

register_family!(
    LOCAL_REQUEST_RETRIES_TOTAL,
    "unkey_frontline_local_request_retries_total",
    "Local-instance requests that hit at least one dial failure, labelled by final outcome.",
    &["outcome"],
    MetricKind::Counter
);

register_family!(
    REGION_FALLBACKS_TOTAL,
    "unkey_frontline_region_fallbacks_total",
    "Requests forwarded to a peer region after every local instance dial-failed.",
    &["to_region"],
    MetricKind::Counter
);

register_family!(
    ROUTING_DECISIONS_TOTAL,
    "unkey_frontline_routing_decisions_total",
    "Routing decisions by type and target region.",
    &["decision", "target_region"],
    MetricKind::Counter
);

register_family!(
    ROUTING_DECISION_SECONDS,
    "unkey_frontline_routing_decision_seconds",
    "Time spent picking a destination for a request.",
    &[],
    MetricKind::Histogram(&[
        0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0
    ])
);

register_family!(
    UPSTREAM_SECONDS,
    "unkey_frontline_upstream_seconds",
    "Upstream call duration.",
    &["destination"],
    MetricKind::Histogram(&[
        0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0
    ])
);

register_family!(
    UPSTREAM_DIALS_TOTAL,
    "unkey_frontline_upstream_dials_total",
    "Upstream dial attempts by destination and outcome.",
    &["destination", "outcome"],
    MetricKind::Counter
);

register_family!(
    HOPS_HISTOGRAM,
    "unkey_frontline_hops",
    "Cross-region hop counts by source and target region.",
    &["src_region", "target_region"],
    MetricKind::Histogram(&[1.0, 2.0, 3.0, 5.0, 10.0])
);

pub fn status_class(code: u16) -> &'static str {
    match code {
        500.. => "5xx",
        400.. => "4xx",
        300.. => "3xx",
        _ => "2xx",
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn counters_and_exposition() {
        REQUESTS_TOTAL.counter_with(&["2xx", "", ""]).inc();
        REQUESTS_TOTAL.counter_with(&["2xx", "", ""]).inc();
        let text = REGISTRY.gather();
        assert!(text.contains("# TYPE unkey_frontline_requests_total counter"));
        assert!(text.contains(
            r#"unkey_frontline_requests_total{status_class="2xx",fault_domain="",code=""} 2"#
        ));
    }

    #[test]
    fn histogram_buckets() {
        let h = ROUTING_DECISION_SECONDS.histogram_with(&[]);
        h.observe(0.0007);
        h.observe(0.3);
        let text = REGISTRY.gather();
        assert!(text.contains("unkey_frontline_routing_decision_seconds_count 2"));
        assert!(text.contains(r#"le="+Inf"} 2"#));
    }

    #[test]
    fn status_classes() {
        assert_eq!(status_class(200), "2xx");
        assert_eq!(status_class(308), "3xx");
        assert_eq!(status_class(404), "4xx");
        assert_eq!(status_class(503), "5xx");
        assert_eq!(status_class(101), "2xx");
    }
}
