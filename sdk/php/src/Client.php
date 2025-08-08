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
        $effective = max(0, min(100, (int)($cfg['all'] ?? 0)));
        foreach ($attrs as $k => $v) {
            if (!isset($cfg['keys'][$k])) continue;
            $kc = $cfg['keys'][$k];
            if (isset($kc['items'][$v])) {
                $effective = max($effective, (int)$kc['items'][$v]);
            } else {
                $effective = max($effective, (int)($kc['all'] ?? 0));
            }
        }
        $enabled = $this->percentageHit($featureName, $seed, $effective);
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
}
