import { IStorage, detectDefaultStorage } from "./storage";

export type KeyConfig = { all: number; items: Record<string, number> };
export type FeatureConfig = {
  name: string;
  all: number;
  keys: Record<string, KeyConfig>;
};

export type UpdateEvent = { version: number; features: FeatureConfig[] };

export type Options = {
  autoSendStats?: boolean;
  onUpdate?: (event: UpdateEvent) => void;
  initialVersion?: number;
  statsFlushIntervalMs?: number;
  pollIntervalMs?: number;
  storage?: IStorage; // custom storage
  storagePrefix?: string; // for browser localStorage
};

type UpdatesRequest = { service_name: string; last_version: number };
type UpdatesResponse = {
  version: number;
  features: Array<{
    all: number;
    name: string;
    props: Array<{ all: number; name: string; item: Record<string, number> }>;
  }>;
  deleted: Array<{
    kind: number;
    feature_name: string;
    key_name?: string;
    param_name?: string;
  }>;
};

export class Client {
  private readonly baseUrl: string;
  private readonly serviceName: string;
  private readonly autoStats: boolean;
  private readonly onUpdate?: (event: UpdateEvent) => void;
  private lastVersion: number;
  private features: Map<string, FeatureConfig> = new Map();
  private stats: Set<string> = new Set();
  private statsTimer: any = null;
  private readonly statsFlushIntervalMs: number;
  private readonly storage: IStorage;
  private pollAbort?: AbortController;

  constructor(baseUrl: string, serviceName: string, opts: Options = {}) {
    if (!baseUrl) throw new Error("baseUrl is required");
    if (!serviceName) throw new Error("serviceName is required");
    this.baseUrl = baseUrl.replace(/\/?$/, "");
    this.serviceName = serviceName;
    this.autoStats = opts.autoSendStats !== false;
    this.onUpdate = opts.onUpdate;
    this.lastVersion = opts.initialVersion ?? 0;
    this.statsFlushIntervalMs = Math.max(
      1000,
      opts.statsFlushIntervalMs ?? 180000
    );
    this.storage =
      opts.storage ?? detectDefaultStorage(opts.storagePrefix ?? "fc");

    // restore from cache if present
    this.restoreCache();
    // start stats flusher
    this.scheduleStatsFlush();
    // auto-start polling in background
    this.pollAbort = new AbortController();
    // best-effort immediate poll
    void this.pollOnce(this.pollAbort.signal);
    // continuous polling
    const every = Math.max(500, opts.pollIntervalMs ?? 3000);
    void this.startPolling(every, this.pollAbort.signal);
  }

  async pollOnce(signal?: AbortSignal): Promise<UpdateEvent | null> {
    const body: UpdatesRequest = {
      service_name: this.serviceName,
      last_version: this.lastVersion,
    };
    const resp = await fetch(this.baseUrl + "/api/updates", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal,
    });
    if (!resp.ok) {
      return null;
    }
    const data = (await resp.json()) as UpdatesResponse;
    this.applyUpdate(data);
    const changed: FeatureConfig[] = [];
    for (const f of data.features) {
      const found = this.features.get(f.name);
      if (found) changed.push(found);
    }
    const event: UpdateEvent = { version: this.lastVersion, features: changed };
    if (this.onUpdate && changed.length > 0) this.onUpdate(event);
    return event;
  }

  close(): void {
    try {
      this.pollAbort?.abort();
    } catch {}
    if (this.statsTimer) {
      clearInterval(this.statsTimer);
      this.statsTimer = null;
    }
  }

  // Simple loop with backoff the caller can await/cancel
  async startPolling(intervalMs = 3000, signal?: AbortSignal): Promise<void> {
    let wait = Math.max(500, intervalMs);
    while (!signal?.aborted) {
      try {
        await this.pollOnce(signal);
        wait = intervalMs;
      } catch {
        wait = Math.min(10000, Math.max(500, Math.floor(wait * 1.5)));
      }
      await new Promise((r) => setTimeout(r, wait));
    }
  }

  isEnabled(
    featureName: string,
    seed: string,
    attrs?: Record<string, string>
  ): boolean {
    const cfg = this.features.get(featureName);
    if (!cfg) return false;
    let percent = -1;
    let keyLevel: number | null = null;
    if (attrs) {
      for (const [k, v] of Object.entries(attrs)) {
        const kc = cfg.keys[k];
        if (!kc) continue;
        if (kc.items[v] !== undefined) {
          percent = kc.items[v];
          break;
        }
        if (keyLevel === null) keyLevel = kc.all;
      }
    }
    if (percent < 0 && keyLevel !== null) {
      percent = keyLevel;
    }
    if (percent < 0) {
      percent = cfg.all;
    }
    percent = clampPercent(percent);
    const enabled = bucketHit(featureName, seed, percent);
    if (enabled && this.autoStats) this.track(featureName);
    return enabled;
  }

  track(featureName: string): void {
    if (featureName) this.stats.add(featureName);
  }

  getSnapshot(): Record<string, FeatureConfig> {
    const out: Record<string, FeatureConfig> = {};
    for (const [k, v] of this.features.entries()) {
      const keys: Record<string, KeyConfig> = {};
      for (const [kk, kv] of Object.entries(v.keys)) {
        keys[kk] = { all: kv.all, items: { ...kv.items } };
      }
      out[k] = { name: v.name, all: v.all, keys };
    }
    return out;
  }

  async flushStats(signal?: AbortSignal): Promise<void> {
    if (this.stats.size === 0) return;
    const list = Array.from(this.stats);
    this.stats.clear();
    try {
      await fetch(this.baseUrl + "/api/stats", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          service_name: this.serviceName,
          features: list,
        }),
        signal,
      });
    } catch {
      // on failure, re-queue (best-effort)
      for (const f of list) this.stats.add(f);
    }
  }

  private scheduleStatsFlush(): void {
    if (this.statsTimer) clearInterval(this.statsTimer);
    this.statsTimer = setInterval(() => {
      void this.flushStats();
    }, this.statsFlushIntervalMs);
  }

  private applyUpdate(data: UpdatesResponse): void {
    // upsert features (respect -1 = no change)
    for (const f of data.features) {
      const name = f.name;
      const existing = this.features.get(name) || {
        name,
        all: 0,
        keys: {} as Record<string, KeyConfig>,
      };
      // Only set top-level percent if not -1
      if (Number(f.all) !== -1) existing.all = Number(f.all);
      if (!existing.keys) existing.keys = {} as Record<string, KeyConfig>;
      for (const p of f.props || []) {
        const keyName = p.name;
        const current = existing.keys[keyName] || {
          all: 0,
          items: {} as Record<string, number>,
        };
        if (Number(p.all) !== -1) current.all = Number(p.all);
        const items = p.item || {};
        for (const [k, v] of Object.entries(items))
          current.items[k] = Number(v);
        existing.keys[keyName] = current;
      }
      this.features.set(name, existing);
    }
    // apply deletions
    for (const d of data.deleted || []) {
      if (d.kind === 0) {
        this.features.delete(d.feature_name);
      } else if (d.kind === 1) {
        const feat = this.features.get(d.feature_name);
        if (feat && d.key_name) delete feat.keys[d.key_name];
      } else if (d.kind === 2) {
        const feat = this.features.get(d.feature_name);
        if (feat && d.key_name && d.param_name)
          delete feat.keys[d.key_name]?.items[d.param_name];
      }
    }
    if (data.version > this.lastVersion) this.lastVersion = data.version;
    this.persistCache();
  }

  private persistCache(): void {
    try {
      const snapshot = this.getSnapshot();
      this.storage.setItem(
        this.cacheKey(),
        JSON.stringify({ version: this.lastVersion, features: snapshot })
      );
    } catch {}
  }

  private restoreCache(): void {
    try {
      const raw = this.storage.getItem(this.cacheKey());
      if (!raw) return;
      const obj = JSON.parse(raw) as {
        version: number;
        features: Record<string, FeatureConfig>;
      } | null;
      if (!obj) return;
      this.lastVersion = obj.version || 0;
      this.features.clear();
      for (const [k, v] of Object.entries(obj.features || {})) {
        this.features.set(k, {
          name: v.name,
          all: Number(v.all),
          keys: v.keys || {},
        });
      }
    } catch {}
  }

  private cacheKey(): string {
    return `fc:${this.serviceName}:v1`;
  }
}

function clampPercent(p: number): number {
  if (p < 0) return 0;
  if (p > 100) return 100;
  return p;
}

function bucketHit(
  featureName: string,
  seed: string,
  percent: number
): boolean {
  if (percent <= 0) return false;
  if (percent >= 100) return true;
  // FNV-1a 64-bit
  let h = 0xcbf29ce484222325n;
  const prime = 0x100000001b3n;
  const s = featureName + "::" + seed;
  for (let i = 0; i < s.length; i++) {
    h ^= BigInt(s.charCodeAt(i));
    h = (h * prime) & 0xffffffffffffffffn;
  }
  const bucket = Number(h % 100n);
  return bucket < percent;
}
