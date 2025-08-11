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

    public function __construct(string $address, string $serviceName, bool $autoSendStats = true)
    {
        if (!$serviceName) { throw new \InvalidArgumentException('serviceName required'); }
        $this->serviceName = $serviceName;
        $this->autoSendStats = $autoSendStats;

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
        // Priority: value match -> key-level -> feature
        $percent = -1;
        foreach ($attrs as $k => $v) {
            if (!isset($cfg['keys'][$k])) continue;
            $kc = $cfg['keys'][$k];
            if (isset($kc['items'][$v])) { $percent = (int)$kc['items'][$v]; break; }
        }
        if ($percent < 0) {
            foreach ($attrs as $k => $_) {
                if (!isset($cfg['keys'][$k])) continue;
                $kc = $cfg['keys'][$k];
                $percent = (int)($kc['all'] ?? 0); break;
            }
        }
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
        // fire-and-forget; best-effort stream send in background could be added later
        $call = $this->statsStub->call();
        $req = new \FeatureChaos\SendStatsRequest();
        $req->setServiceName($this->serviceName);
        $req->setFeatureName($featureName);
        $call->write($req);
        $call->writesDone();
        $call->getStatus();
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
        $this->lastVersion = max($this->lastVersion, (int)$resp->getVersion());
    }

    private function percentageHit(string $featureName, string $seed, int $percent): bool
    {
        $percent = max(0, min(100, $percent));
        if ($percent <= 0) return false;
        if ($percent >= 100) return true;
        $h = crc32($featureName . '::' . $seed);
        return ($h % 100) < $percent;
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
}
