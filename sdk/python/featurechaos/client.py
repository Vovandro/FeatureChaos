import contextlib
import hashlib
import threading
import time
from dataclasses import dataclass
from typing import Callable, Dict, Optional

import grpc
from google.protobuf import empty_pb2
from google.protobuf import descriptor_pb2
from google.protobuf import descriptor_pool
from google.protobuf import message_factory


@dataclass
class Options:
    auto_send_stats: bool = True
    on_update: Optional[Callable[[int, Dict[str, dict]], None]] = None
    initial_version: int = 0


class FeatureChaosClient:
    def __init__(self, address: str, service_name: str, options: Options = Options()):
        if not address or ":" not in address:
            raise ValueError("address must be host:port")
        if not service_name:
            raise ValueError("service_name is required")
        self._service_name = service_name
        self._options = options
        self._last_version = options.initial_version
        self._features = {}  # name -> {"all": int, "keys": {key: {"all": int, "items": {val: int}}}}
        self._lock = threading.RLock()
        self._stats_queue = []
        self._stop = threading.Event()

        # gRPC channel
        self._channel = grpc.insecure_channel(address)

        # Build dynamic stubs from descriptors (keeps SDK standalone)
        self._pool = descriptor_pool.Default()
        self._factory = message_factory.MessageFactory(self._pool)
        self._build_descriptors()

        # Start background threads
        self._subscriber_thread = threading.Thread(target=self._run_subscriber, daemon=True)
        self._subscriber_thread.start()
        self._stats_thread = threading.Thread(target=self._run_stats, daemon=True)
        self._stats_thread.start()

    def close(self):
        self._stop.set()
        self._subscriber_thread.join(timeout=2)
        self._stats_thread.join(timeout=2)
        with contextlib.suppress(Exception):
            self._channel.close()

    def is_enabled(self, feature_name: str, seed: str, attrs: Dict[str, str]) -> bool:
        with self._lock:
            cfg = self._features.get(feature_name)
        if not cfg:
            return False

        keys = cfg.get("keys", {})
        percent = -1
        # pass 1: any exact value match among provided attrs
        for k, v in (attrs or {}).items():
            kc = keys.get(k)
            if kc and v in kc.get("items", {}):
                percent = int(kc["items"][v])
                break
        # pass 2: any key-level percent if no exact match
        if percent < 0:
            for k in (attrs or {}).keys():
                kc = keys.get(k)
                if kc:
                    percent = int(kc.get("all", 0))
                    break
        if percent < 0:
            percent = int(cfg.get("all", 0))

        if percent <= 0:
            return False
        if percent >= 100:
            enabled = True
        else:
            enabled = self._fast_bucket_hit(feature_name, seed, percent)
        if enabled and self._options.auto_send_stats:
            self.track(feature_name)
        return enabled

    def track(self, feature_name: str):
        with self._lock:
            if len(self._stats_queue) < 1000:
                self._stats_queue.append(feature_name)

    def get_snapshot(self) -> Dict[str, dict]:
        with self._lock:
            # deep-ish copy
            return {k: {"all": v["all"], "keys": {kk: {"all": vv["all"], "items": dict(vv["items"]) } for kk, vv in v["keys"].items()}} for k, v in self._features.items()}

    # internals
    def _build_descriptors(self):
        # Construct messages similar to proto/FeatureChaos.proto
        file_desc = descriptor_pb2.FileDescriptorProto()
        file_desc.name = "FeatureChaos.proto"
        file_desc.package = "FeatureChaos"

        # PropsItem
        props = file_desc.message_type.add()
        props.name = "PropsItem"
        f = props.field.add(); f.name = "All"; f.number = 1; f.label = 1; f.type = 5
        f = props.field.add(); f.name = "Name"; f.number = 2; f.label = 1; f.type = 9
        f = props.field.add(); f.name = "Item"; f.number = 3; f.label = 3; f.type = 11; f.type_name = ".FeatureChaos.PropsItem.ItemEntry"
        entry = props.nested_type.add(); entry.name = "ItemEntry"; entry.options.map_entry = True
        f = entry.field.add(); f.name = "key"; f.number = 1; f.label = 1; f.type = 9
        f = entry.field.add(); f.name = "value"; f.number = 2; f.label = 1; f.type = 5

        # FeatureItem
        feat = file_desc.message_type.add()
        feat.name = "FeatureItem"
        f = feat.field.add(); f.name = "All"; f.number = 1; f.label = 1; f.type = 5
        f = feat.field.add(); f.name = "Name"; f.number = 2; f.label = 1; f.type = 9
        f = feat.field.add(); f.name = "Props"; f.number = 3; f.label = 3; f.type = 11; f.type_name = ".FeatureChaos.PropsItem"

        # GetAllFeatureRequest
        req = file_desc.message_type.add()
        req.name = "GetAllFeatureRequest"
        f = req.field.add(); f.name = "ServiceName"; f.number = 1; f.label = 1; f.type = 9
        f = req.field.add(); f.name = "LastVersion"; f.number = 2; f.label = 1; f.type = 3

        # SendStatsRequest
        stats = file_desc.message_type.add()
        stats.name = "SendStatsRequest"
        f = stats.field.add(); f.name = "ServiceName"; f.number = 1; f.label = 1; f.type = 9
        f = stats.field.add(); f.name = "FeatureName"; f.number = 2; f.label = 1; f.type = 9

        # GetFeatureResponse
        resp = file_desc.message_type.add()
        resp.name = "GetFeatureResponse"
        f = resp.field.add(); f.name = "Version"; f.number = 1; f.label = 1; f.type = 3
        f = resp.field.add(); f.name = "Features"; f.number = 2; f.label = 3; f.type = 11; f.type_name = ".FeatureChaos.FeatureItem"

        # Service
        svc = file_desc.service.add(); svc.name = "FeatureService"
        m = svc.method.add(); m.name = "Subscribe"; m.input_type = ".FeatureChaos.GetAllFeatureRequest"; m.output_type = ".FeatureChaos.GetFeatureResponse"; m.server_streaming = True
        m = svc.method.add(); m.name = "Stats"; m.input_type = ".FeatureChaos.SendStatsRequest"; m.output_type = ".google.protobuf.Empty"; m.client_streaming = True

        # Import google empty
        file_desc.dependency.append("google/protobuf/empty.proto")

        self._pool.Add(file_desc)

        self._Req = self._factory.GetPrototype(self._pool.FindMessageTypeByName("FeatureChaos.GetAllFeatureRequest"))
        self._Resp = self._factory.GetPrototype(self._pool.FindMessageTypeByName("FeatureChaos.GetFeatureResponse"))
        self._StatsReq = self._factory.GetPrototype(self._pool.FindMessageTypeByName("FeatureChaos.SendStatsRequest"))

        self._subscribe = self._channel.unary_stream(
            "/FeatureChaos.FeatureService/Subscribe",
            request_serializer=self._Req.SerializeToString,
            response_deserializer=self._Resp.FromString,
        )
        self._stats = self._channel.stream_unary(
            "/FeatureChaos.FeatureService/Stats",
            request_serializer=self._StatsReq.SerializeToString,
            response_deserializer=empty_pb2.Empty.FromString,
        )

    def _run_subscriber(self):
        backoff = 0.5
        while not self._stop.is_set():
            try:
                req = self._Req(ServiceName=self._service_name, LastVersion=self._last_version)
                stream = self._subscribe(req, timeout=30)
                backoff = 0.5
                for resp in stream:
                    version = getattr(resp, "Version", 0)
                    features = getattr(resp, "Features", [])
                    self._apply_update(version, features)
            except Exception:
                time.sleep(min(10.0, backoff))
                backoff = min(10.0, backoff * 2)

    def _apply_update(self, version: int, features):
        with self._lock:
            for f in features:
                name = getattr(f, "Name", "")
                allp = int(getattr(f, "All", 0))
                props = getattr(f, "Props", [])
                keys = {}
                for p in props:
                    kname = getattr(p, "Name", "")
                    kall = int(getattr(p, "All", 0))
                    items = dict(getattr(p, "Item", {}))
                    keys[kname] = {"all": kall, "items": {str(k): int(v) for k, v in items.items()}}
                self._features[name] = {"all": allp, "keys": keys}
            if version > self._last_version:
                self._last_version = version
        cb = self._options.on_update
        if cb:
            cb(self._last_version, self.get_snapshot())

    def _run_stats(self):
        while not self._stop.is_set():
            # drain queue chunk
            with self._lock:
                if not self._stats_queue:
                    time.sleep(0.2)
                    continue
                chunk = self._stats_queue[:100]
                self._stats_queue = self._stats_queue[100:]
            # send stream
            with contextlib.suppress(Exception):
                call = self._stats()
                for feat in chunk:
                    req = self._StatsReq(ServiceName=self._service_name, FeatureName=feat)
                    call.write(req)
                call.done_writing()
                _ = call.result()

    @staticmethod
    def _percentage_hit(feature_name: str, seed: str, percent: int) -> bool:
        percent = max(0, min(100, int(percent)))
        if percent <= 0:
            return False
        if percent >= 100:
            return True
        h = hashlib.blake2b(digest_size=8)
        h.update(feature_name.encode()); h.update(b"::"); h.update(seed.encode())
        bucket = int.from_bytes(h.digest(), "big") % 100
        return bucket < percent

    @staticmethod
    def _fast_bucket_hit(feature_name: str, seed: str, percent: int) -> bool:
        # FNV-1a 64-bit, minimal allocations
        # clamp already checked by caller
        h = 0xcbf29ce484222325
        for ch in feature_name.encode():
            h ^= ch
            h = (h * 0x100000001b3) & 0xFFFFFFFFFFFFFFFF
        # '::'
        h ^= ord(':'); h = (h * 0x100000001b3) & 0xFFFFFFFFFFFFFFFF
        h ^= ord(':'); h = (h * 0x100000001b3) & 0xFFFFFFFFFFFFFFFF
        for ch in seed.encode():
            h ^= ch
            h = (h * 0x100000001b3) & 0xFFFFFFFFFFFFFFFF
        bucket = h % 100
        return bucket < percent
