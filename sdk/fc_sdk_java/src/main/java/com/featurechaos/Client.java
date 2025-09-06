package com.featurechaos;

import FeatureChaos.FeatureServiceGrpc;
import FeatureChaos.FeatureServiceGrpc.FeatureServiceBlockingStub;
import FeatureChaos.FeatureServiceGrpc.FeatureServiceStub;
import FeatureChaos.GetAllFeatureRequest;
import FeatureChaos.GetFeatureResponse;
import FeatureChaos.SendStatsRequest;
import com.google.protobuf.Empty;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.ClientCallStreamObserver;
import io.grpc.stub.ClientResponseObserver;
import io.grpc.stub.StreamObserver;

import java.time.Duration;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

/** Java SDK client mirroring Go SDK behavior. */
public class Client {
    public static class KeyConfig { public int all; public Map<String, Integer> items = new HashMap<>(); }
    public static class FeatureConfig { public String name; public int all; public Map<String, KeyConfig> keys = new HashMap<>(); }
    public static class UpdateEvent { public long version; public Map<String, FeatureConfig> features; }

    private final ManagedChannel channel;
    private final FeatureServiceStub async;
    private final String serviceName;
    private final boolean autoSendStats;
    private final Consumer<UpdateEvent> onUpdate;
    private volatile long lastVersion;
    private final Map<String, FeatureConfig> features = new HashMap<>();
    private final Set<String> stats = Collections.synchronizedSet(new HashSet<>());
    private final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(2);
    private final long statsFlushMs;

    public static class Options {
        public boolean autoSendStats = true;
        public Consumer<UpdateEvent> onUpdate;
        public long initialVersion = 0;
        public Duration statsFlushInterval = Duration.ofMinutes(3);
    }

    public Client(String address, String serviceName, Options options) {
        Objects.requireNonNull(address, "address");
        Objects.requireNonNull(serviceName, "serviceName");
        if (options == null) options = new Options();

        this.channel = ManagedChannelBuilder.forTarget(address).usePlaintext().build();
        this.async = FeatureServiceGrpc.newStub(channel);
        this.serviceName = serviceName;
        this.autoSendStats = options.autoSendStats;
        this.onUpdate = options.onUpdate;
        this.lastVersion = options.initialVersion;
        this.statsFlushMs = Math.max(1000, options.statsFlushInterval.toMillis());

        // auto-start subscriber and stats flusher
        startSubscriber();
        scheduler.scheduleAtFixedRate(this::flushStatsSafe, this.statsFlushMs, this.statsFlushMs, TimeUnit.MILLISECONDS);
    }

    public void close() {
        try { channel.shutdownNow().awaitTermination(2, TimeUnit.SECONDS); } catch (InterruptedException ignored) {}
        scheduler.shutdownNow();
    }

    public boolean isEnabled(String featureName, String seed, Map<String, String> attrs) {
        FeatureConfig cfg;
        synchronized (features) { cfg = features.get(featureName); }
        if (cfg == null) return false;
        int percent = -1; Integer keyLevel = null;
        if (attrs != null) {
            for (Map.Entry<String, String> e : attrs.entrySet()) {
                KeyConfig kc = cfg.keys.get(e.getKey());
                if (kc == null) continue;
                Integer p = kc.items.get(e.getValue());
                if (p != null) { percent = p; break; }
                if (keyLevel == null) keyLevel = kc.all;
            }
        }
        if (percent < 0 && keyLevel != null) { percent = keyLevel; }
        if (percent < 0) { percent = cfg.all; }
        percent = clampPercent(percent);
        boolean enabled = bucketHit(featureName, seed, percent);
        if (enabled && autoSendStats) track(featureName);
        return enabled;
    }

    public void track(String featureName) { if (featureName != null && !featureName.isEmpty()) stats.add(featureName); }

    public Map<String, FeatureConfig> getSnapshot() {
        synchronized (features) { return new HashMap<>(features); }
    }

    private void startSubscriber() {
        backoffLoop(500, 10000, () -> {
            GetAllFeatureRequest req = GetAllFeatureRequest.newBuilder()
                    .setServiceName(serviceName)
                    .setLastVersion(lastVersion)
                    .build();

            async.subscribe(req, new StreamObserver<>() {
                @Override public void onNext(GetFeatureResponse resp) { applyUpdate(resp); }
                @Override public void onError(Throwable t) { /* reconnect via backoff */ throw new RuntimeException(t); }
                @Override public void onCompleted() { throw new RuntimeException("completed"); }
            });
        });
    }

    private void backoffLoop(long initialMs, long maxMs, Runnable connect) {
        scheduler.execute(() -> {
            long wait = initialMs;
            while (!channel.isShutdown()) {
                try {
                    connect.run();
                    // if connect returns (stream completed), wait and retry
                } catch (Throwable ignored) {}
                try { Thread.sleep(wait); } catch (InterruptedException ignored) {}
                wait = Math.min(maxMs, Math.max(initialMs, (long)(wait * 1.5)));
            }
        });
    }

    private void applyUpdate(GetFeatureResponse resp) {
        synchronized (features) {
            // upsert with -1 meaning no change for all values
            for (FeatureChaos.FeatureItem f : resp.getFeaturesList()) {
                String name = f.getName();
                FeatureConfig cfg = features.getOrDefault(name, new FeatureConfig());
                cfg.name = name;
                if (cfg.keys == null) cfg.keys = new HashMap<>();
                if (f.getAll() != -1) cfg.all = f.getAll();
                for (FeatureChaos.PropsItem p : f.getPropsList()) {
                    KeyConfig kc = cfg.keys.getOrDefault(p.getName(), new KeyConfig());
                    if (p.getAll() != -1) kc.all = p.getAll();
                    kc.items.putAll(p.getItemMap());
                    cfg.keys.put(p.getName(), kc);
                }
                features.put(name, cfg);
            }
            // deletions
            for (FeatureChaos.GetFeatureResponse.DeletedItem d : resp.getDeletedList()) {
                switch (d.getKind()) {
                    case FEATURE -> features.remove(d.getFeatureName());
                    case KEY -> { FeatureConfig fc = features.get(d.getFeatureName()); if (fc != null) fc.keys.remove(d.getKeyName()); }
                    case PARAM -> { FeatureConfig fc = features.get(d.getFeatureName()); if (fc != null) { KeyConfig kc = fc.keys.get(d.getKeyName()); if (kc != null) kc.items.remove(d.getParamName()); } }
                    default -> {}
                }
            }
            if (resp.getVersion() > lastVersion) lastVersion = resp.getVersion();
        }
        if (onUpdate != null) {
            UpdateEvent ev = new UpdateEvent();
            ev.version = lastVersion;
            ev.features = getSnapshot();
            try { onUpdate.accept(ev); } catch (Exception ignored) {}
        }
    }

    private void flushStatsSafe() {
        try { flushStats(); } catch (Exception ignored) {}
    }

    public void flushStats() {
        if (stats.isEmpty()) return;
        Set<String> toSend = new HashSet<>(stats); stats.clear();
        FeatureServiceBlockingStub blocking = FeatureServiceGrpc.newBlockingStub(channel);
        io.grpc.stub.StreamObserver<Empty> close = new StreamObserver<>() {
            @Override public void onNext(Empty value) {}
            @Override public void onError(Throwable t) {}
            @Override public void onCompleted() {}
        };
        ClientResponseObserver<SendStatsRequest, Empty> req = new ClientResponseObserver<>() {
            ClientCallStreamObserver<SendStatsRequest> requestStream;
            @Override public void beforeStart(ClientCallStreamObserver<SendStatsRequest> rs) { this.requestStream = rs; }
            @Override public void onNext(Empty empty) {}
            @Override public void onError(Throwable throwable) {}
            @Override public void onCompleted() {}
        };
        FeatureServiceStub stub = FeatureServiceGrpc.newStub(channel);
        StreamObserver<SendStatsRequest> stream = stub.stats(new StreamObserver<>() {
            @Override public void onNext(Empty value) {}
            @Override public void onError(Throwable t) {}
            @Override public void onCompleted() {}
        });
        for (String name : toSend) {
            SendStatsRequest reqMsg = SendStatsRequest.newBuilder().setServiceName(serviceName).setFeatureName(name).build();
            try { stream.onNext(reqMsg); } catch (StatusRuntimeException e) { break; }
        }
        try { stream.onCompleted(); } catch (Exception ignored) {}
    }

    private static int clampPercent(int p) { if (p < 0) return 0; if (p > 100) return 100; return p; }
    private static boolean bucketHit(String featureName, String seed, int percent) {
        if (percent <= 0) return false; if (percent >= 100) return true;
        long[] box = fnv64(featureName + "::" + seed);
        long hash = box[0];
        int bucket = (int) (hash % 100L);
        return bucket < percent;
    }
    private static long[] fnv64(String s) {
        long hash = 0xcbf29ce484222325L;
        long prime = 0x100000001b3L;
        for (int i = 0; i < s.length(); i++) {
            hash ^= (long) s.charAt(i);
            hash = (hash * prime) & 0xffffffffffffffffL;
        }
        return new long[]{hash};
    }
}
