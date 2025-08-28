FeatureChaos HTTP SDK (JS/TS)

A lightweight client that polls FeatureChaos HTTP public endpoints and evaluates feature flags locally. Works in browsers and Node.js.

By default, the client auto-polls for updates and auto-sends stats.

Install

- TypeScript/ESM: copy `ts/` sources into your project and build
- Plain JS: use `js/client.js`

Usage (TypeScript)

```ts
import { Client } from "./ts/client";

const client = new Client("https://fc.example.com", "my-service", {
  autoSendStats: true,
  initialVersion: 0,
  storagePrefix: "fc",
  pollIntervalMs: 3000,
  onUpdate: (ev) => console.log("updated to", ev.version),
});

// auto-polling starts immediately
const enabled = client.isEnabled("my_feature", "user_123", { country: "US" });
if (enabled) client.track("my_feature");

// on shutdown
client.close();
```

Usage (Browser JS)

```html
<script src="/path/to/js/client.js"></script>
<script>
  const client = new window.FeatureChaosHttpClient("/", "my-service");
  // auto-polling starts immediately
  function check() {
    const on = client.isEnabled("my_feature", "user42", { plan: "pro" });
    console.log("flag:", on);
  }
  check();
  // client.close(); // on page unload if needed
</script>
```

API

- new Client(baseUrl, serviceName, opts?)
  - baseUrl: server base URL for public HTTP
  - serviceName: name used for segmentation and stats
  - opts.autoSendStats (default true)
  - opts.onUpdate?: (event) => void
  - opts.initialVersion?: number
  - opts.statsFlushIntervalMs?: number (default 180000)
  - opts.pollIntervalMs?: number (default 3000)
  - opts.storage?: IStorage custom storage
  - opts.storagePrefix?: string prefix for browser localStorage
- Polling and stats flushing start automatically
- startPolling(intervalMs=3000, signal?) — optional manual control
- pollOnce(signal?) — optional single fetch
- isEnabled(name, seed, attrs?)
- track(name)
- getSnapshot()
- flushStats(signal?)
- close()

Storage

By default the SDK uses localStorage in browsers and in-memory storage in Node.js. You can provide a custom storage implementing { getItem, setItem, removeItem }.
