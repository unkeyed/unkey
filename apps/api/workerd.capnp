using Workerd = import "/workerd/workerd.capnp";

const config :Workerd.Config = (
  services = [
    (name = "main", worker = .mainWorker),

  ],

  sockets = [
    # Serve HTTP on port 8787.
    ( name = "http",
      address = "*:8787",
      http = (),
      service = "main"
    ),
  ]
);

const mainWorker :Workerd.Worker = (
modules = [
    (name = "worker", esModule = embed "dist/worker.js")
  ],

  durableObjectNamespaces = [
    (className = "DO_RATELIMIT", uniqueKey = "ratelimit"),
    (className = "DO_USAGELIMIT", uniqueKey = "usagelimit"),
  ],
    durableObjectStorage = (inMemory = void),

bindings = [
    (name = "DO_RATELIMIT", durableObjectNamespace = "DO_RATELIMIT"),
    (name = "DO_USAGELIMIT", durableObjectNamespace = "DO_USAGELIMIT"),
    ( name = "DATABASE_HOST", fromEnvironment= "DATABASE_HOST"),
    ( name = "DATABASE_USERNAME", fromEnvironment= "DATABASE_USERNAME"),
    ( name = "DATABASE_PASSWORD", fromEnvironment= "DATABASE_PASSWORD"),
  ],

  compatibilityDate = "2023-02-28",
  # Learn more about compatibility dates at:
  # https://developers.cloudflare.com/workers/platform/compatibility-dates/
);