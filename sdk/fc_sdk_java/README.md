FeatureChaos Java SDK (gRPC)

Gradle Java library that connects to FeatureChaos gRPC server, subscribes for updates, evaluates flags locally, and client-streams stats.

Proto sources are consumed directly from the repository root (`proto/FeatureChaos.proto`), duplication is not needed.

Build

```
./gradlew :sdk:fc_sdk_java:build
```

Usage

```java
import com.featurechaos.Client;

Client.Options opts = new Client.Options();
opts.autoSendStats = true;
Client c = new Client("localhost:50051", "my-service", opts);

boolean enabled = c.isEnabled("my_feature", "user_123", Map.of("country", "US"));
if (enabled) c.track("my_feature");

// shutdown
c.close();
```
