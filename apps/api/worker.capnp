using Workerd = import "/workerd/workerd.capnp";



const mainWorker :Workerd.Worker = (
  globalOutbound = "fullNetwork",
  modules = [
    (name = "worker", esModule = embed "./dist/worker.js")
  ],

  

  durableObjectNamespaces = [
    (className = "DurableObjectRatelimiter", uniqueKey = "ratelimit"),
    (className = "DurableObjectUsagelimiter", uniqueKey = "usagelimit"),
  ],
    durableObjectStorage = (inMemory = void),

bindings = [
    ( name = "DO_RATELIMIT", durableObjectNamespace = "DurableObjectRatelimiter"),
    ( name = "DO_USAGELIMIT", durableObjectNamespace = "DurableObjectUsagelimiter"),
    ( name = "KEY_MIGRATIONS", queue = "main"),
    ( name = "DATABASE_HOST", fromEnvironment= "DATABASE_HOST"),
    ( name = "DATABASE_USERNAME", fromEnvironment= "DATABASE_USERNAME"),
    ( name = "DATABASE_PASSWORD", fromEnvironment= "DATABASE_PASSWORD"),
    ( name = "AGENT_URL", fromEnvironment= "AGENT_URL"),
    ( name = "AGENT_TOKEN", fromEnvironment= "AGENT_TOKEN"),
    ( name = "VERSION", fromEnvironment= "VERSION"),
  ],

  compatibilityDate = "2024-02-19",
  compatibilityFlags = ["nodejs_compat"]
  # Learn more about compatibility dates at:
  # https://developers.cloudflare.com/workers/platform/compatibility-dates/

  
);

const config :Workerd.Config = (
  services = [
    (
      name = "main", 
      worker = .mainWorker,
    ),
    (
      name = "fullNetwork",
      network = (
  allow = ["public", "private", "local", "network", "unix", "unix-abstract"],
  tlsOptions = (trustBrowserCas = true)
)
    )
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
