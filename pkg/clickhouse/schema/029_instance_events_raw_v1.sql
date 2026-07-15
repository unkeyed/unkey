-- Instance lifecycle events: container running, termination, and waiting
-- (CrashLoopBackOff today; ImagePullBackOff, etc. later) transitions
-- captured by krane's pod watch. Mirrors corev1.ContainerState. Surfaces
-- user-actionable failures (OOMKilled, exit codes, crashloops) the gateway
-- can't currently report, and powers the logs viewer's lifecycle dividers
-- (running → terminated, with waiting events between).

CREATE TABLE IF NOT EXISTS default.instance_events_raw_v1
(
    -- When this event happened (unix milliseconds). For 'running' it's the
    -- container's StartedAt; for 'terminated' it's FinishedAt; for
    -- 'waiting' it's when kubelet last published the waiting state.
    -- Same encoding as runtime_logs_raw_v1.time.
    `time` Int64 CODEC(Delta, LZ4),

    -- Customer identifiers (extracted from pod labels by krane). Column
    -- order matches the canonical hierarchy: workspace > project > app >
    -- environment > deployment.
    `workspace_id`   String CODEC(ZSTD(1)),
    `project_id`     String CODEC(ZSTD(1)),
    `app_id`         String CODEC(ZSTD(1)),
    `environment_id` String CODEC(ZSTD(1)),
    `deployment_id`  String CODEC(ZSTD(1)),

    -- K8s identity. Each (pod_uid, container_name, restart_count) is one
    -- container life; the LRU dedupe key in krane uses the same tuple plus
    -- event_kind, so a life can produce one running + one terminated row
    -- without colliding.
    `pod_uid`        String CODEC(ZSTD(1)),
    `pod_name`       String CODEC(ZSTD(1)),
    `node_name`      String CODEC(ZSTD(1)),
    `container_name` String CODEC(ZSTD(1)),
    `container_id`   String CODEC(ZSTD(1)),
    `restart_count`  Int32,

    -- Discriminator derived from the proto's `state` oneof case:
    --   'running'    — corev1.ContainerStateRunning
    --   'terminated' — corev1.ContainerStateTerminated
    --   'waiting'    — corev1.ContainerStateWaiting (reason carries the
    --                  specific waiting cause: CrashLoopBackOff, etc.).
    `event_kind`     LowCardinality(String),

    -- Kubelet-supplied state metadata. exit_code/signal are populated for
    -- 'terminated' only; reason/message are populated for 'terminated' and
    -- 'waiting'. All zero/empty for 'running'.
    `exit_code`      Int32,
    `signal`         Int32,
    `reason`         LowCardinality(String),
    `message`        String CODEC(ZSTD(1)),

    `region`         LowCardinality(String),
    `platform`       LowCardinality(String),

    -- Stable hash of (image_id, exit_code, reason, message[:200]). Lets the
    -- dashboard group "you've had this same OOM 17 times" without an
    -- aggregate table.
    `event_fingerprint` String CODEC(ZSTD(1)),

    -- Selected k8s metadata captured from the pod at event time. JSON keeps
    -- the schema stable as we add new context fields without ALTER. Known
    -- keys (krane is the source of truth):
    --   image, image_id           — what code was running
    --   cpu_limit_millicores      — Resources.Limits.Cpu()
    --   memory_limit_mib          — Resources.Limits.Memory()
    --   cpu_request_millicores    — Resources.Requests.Cpu()
    --   memory_request_mib        — Resources.Requests.Memory()
    --   build_id                  — unkey.com/build.id label
    -- ctrl serializes the proto map<string,string> into a JSON string
    -- before insert; an empty map becomes "{}", which the column parses.
    `attributes` JSON CODEC(ZSTD(1)),
    `attributes_text` String MATERIALIZED toJSONString(attributes) CODEC(ZSTD(1)),

    -- Dashboard hot path is WHERE deployment_id = ? ORDER BY time DESC.
    -- The ORDER BY tuple prunes parts by workspace/project/env/time; bloom
    -- filters narrow remaining granules by the secondary id.
    INDEX idx_deployment_id     deployment_id     TYPE bloom_filter(0.001) GRANULARITY 1,
    -- Logs viewer joins by pod_uid in a small time window.
    INDEX idx_pod_uid           pod_uid           TYPE bloom_filter(0.001) GRANULARITY 1,
    -- Incident grouping: GROUP BY event_fingerprint.
    INDEX idx_event_fingerprint event_fingerprint TYPE bloom_filter(0.001) GRANULARITY 1,
    -- Events panel kind filter: 'running' | 'terminated' | 'waiting'. The
    -- query uses `event_kind IN (...)` so a set index prunes granules.
    INDEX idx_event_kind        event_kind        TYPE set(16) GRANULARITY 1,
    -- ngram bloom filter on lower(message) so positionCaseInsensitive(lower(message), lower(...))
    -- can use it. tokenbf_v1 only matches whole tokens and never helped substring search;
    -- this matches the pattern in runtime_logs_raw_v1.
    INDEX idx_message_text_search lower(message)  TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toDate(fromUnixTimestamp64Milli(time))
ORDER BY (workspace_id, project_id, app_id, environment_id, time, deployment_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000, ttl_only_drop_parts = 1;
