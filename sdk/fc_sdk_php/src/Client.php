<?php

namespace FeatureChaos;

use Grpc\ChannelCredentials;
use Google\Protobuf\GPBEmpty;

class Client
{
    private \Grpc\BaseStub $subscribeStub;
    private \Grpc\BaseStub $statsStub;
    private string $serviceName;
    private int $lastVersion = 0;
    private array $features = [];
    private bool $autoSendStats = true;
    private array $statsBuffer = [];
    private float $lastFlush = 0.0;
    private float $flushIntervalSec = 180.0; // default ~3 minutes

    public function __construct(string $address, string $serviceName, bool $autoSendStats = true, ?float $flushIntervalSec = null)
    {
        if (!$serviceName) { throw new \InvalidArgumentException('serviceName required'); }
        $this->serviceName = $serviceName;
        $this->autoSendStats = $autoSendStats;
        $this->lastFlush = microtime(true);
        if ($flushIntervalSec !== null && $flushIntervalSec > 0) {
            $this->flushIntervalSec = $flushIntervalSec;
        }

        // Low-level dynamic stubs using generic grpc BaseStub
        $this->subscribeStub = new class($address, ['credentials' => ChannelCredentials::createInsecure()]) extends \Grpc\BaseStub {
            public function call($arg)
            {
                return $this->_serverStreamRequest('/FeatureChaos.FeatureService/Subscribe', $arg, ['\FeatureChaos\GetFeatureResponse', 'decode']);
            }
        };
        $this->statsStub = new class($address, ['credentials' => ChannelCredentials::createInsecure()]) extends \Grpc\BaseStub {
            public function call()
            {
                return $this->_clientStreamRequest('/FeatureChaos.FeatureService/Stats', ['Google\\Protobuf\\GPBEmpty', 'decode']);
            }
        };
    }

    public function isEnabled(string $featureName, string $seed, array $attrs): bool
    {
        $cfg = $this->features[$featureName] ?? null;
        if (!$cfg) return false;
        // Priority & seeding rules:
        // - exact key/value match -> use the attribute value as hash seed
        // - key exists but no value match -> use provided seed
        // - no key match -> use provided seed with feature-level percent
        $percent = -1;
        $keyLevel = null;
        foreach (($attrs ?? []) as $k => $v) {
            if (!isset($cfg['keys'][$k])) continue;
            $kc = $cfg['keys'][$k];
            if (isset($kc['items'][$v])) { $percent = (int)$kc['items'][$v]; break; }
            if ($keyLevel === null) { $keyLevel = (int)($kc['all'] ?? 0); }
        }
        if ($percent < 0 && $keyLevel !== null) { $percent = $keyLevel; }
        if ($percent < 0) { $percent = (int)($cfg['all'] ?? 0); }
        if ($percent <= 0) return false;
        if ($percent >= 100) $enabled = true; else $enabled = $this->fastBucketHit($featureName, $seed, $percent);
        if ($enabled && $this->autoSendStats) {
            $this->track($featureName);
        }
        return $enabled;
    }

    public function track(string $featureName): void
    {
        // buffer and flush no more than every few minutes
        $this->statsBuffer[$featureName] = ($this->statsBuffer[$featureName] ?? 0) + 1;
        $now = microtime(true);
        if ($now - $this->lastFlush >= $this->flushIntervalSec || count($this->statsBuffer) > 500) {
            $this->flushStats();
        }
    }

    public function flushStats(): void
    {
        if (empty($this->statsBuffer)) return;
        $buf = $this->statsBuffer; // copy
        $this->statsBuffer = [];
        $this->lastFlush = microtime(true);
        try {
            $call = $this->statsStub->call();
            foreach ($buf as $name => $_n) {
                // send at most one event per feature per interval
                $req = new \FeatureChaos\SendStatsRequest();
                $req->setServiceName($this->serviceName);
                $req->setFeatureName($name);
                $call->write($req);
            }
            $call->writesDone();
            $call->getStatus();
        } catch (\Throwable $e) {
            // swallow errors; best-effort
        }
    }

    public function startSubscribeLoop(): void
    {
        $req = new \FeatureChaos\GetAllFeatureRequest();
        $req->setServiceName($this->serviceName);
        $req->setLastVersion($this->lastVersion);
        $stream = $this->subscribeStub->call($req);
        while ($resp = $stream->read()) {
            $this->applyUpdate($resp);
        }
    }

    private function applyUpdate(\FeatureChaos\GetFeatureResponse $resp): void
    {
        foreach ($resp->getFeatures() as $f) {
            $name = $f->getName();
            $cfg = [
                'all' => (int)$f->getAll(),
                'keys' => []
            ];
            foreach ($f->getProps() as $p) {
                $cfg['keys'][$p->getName()] = [
                    'all' => (int)$p->getAll(),
                    'items' => iterator_to_array($p->getItem())
                ];
            }
            $this->features[$name] = $cfg;
        }
        // Apply deletions after feature updates
        foreach ($resp->getDeleted() as $d) {
            $kind = $d->getKind();
            if ($kind === \FeatureChaos\GetFeatureResponse\DeletedItem\Type::FEATURE) {
                unset($this->features[$d->getFeatureName()]);
                continue;
            }
            if ($kind === \FeatureChaos\GetFeatureResponse\DeletedItem\Type::KEY) {
                $fn = $d->getFeatureName();
                $kn = $d->getKeyName();
                if (isset($this->features[$fn]['keys'][$kn])) {
                    unset($this->features[$fn]['keys'][$kn]);
                }
                continue;
            }
            if ($kind === \FeatureChaos\GetFeatureResponse\DeletedItem\Type::PARAM) {
                $fn = $d->getFeatureName();
                $kn = $d->getKeyName();
                $pn = $d->getParamName();
                if (isset($this->features[$fn]['keys'][$kn]['items'][$pn])) {
                    unset($this->features[$fn]['keys'][$kn]['items'][$pn]);
                }
                continue;
            }
        }
        $this->lastVersion = max($this->lastVersion, (int)$resp->getVersion());
    }

    private function fastBucketHit(string $featureName, string $seed, int $percent): bool
    {
        // FNV-1a 64-bit in PHP ints
        $h = 0xcbf29ce484222325;
        $a = $featureName . '::' . $seed;
        $len = strlen($a);
        for ($i=0; $i<$len; $i++) {
            $h ^= ord($a[$i]);
            // multiply modulo 2^64; PHP int might be 64-bit on most envs; keep mask
            $h = ($h * 0x100000001b3) & 0xFFFFFFFFFFFFFFFF;
        }
        $bucket = $h % 100;
        return $bucket < $percent;
    }

    public function __destruct()
    {
        // best-effort final flush of pending stats
        $this->flushStats();
    }
}
