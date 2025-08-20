<?php
// GENERATED CODE -- DO NOT EDIT!

namespace FeatureChaos;

/**
 */
class FeatureServiceClient extends \Grpc\BaseStub {

    /**
     * @param string $hostname hostname
     * @param array $opts channel options
     * @param \Grpc\Channel $channel (optional) re-use channel object
     */
    public function __construct($hostname, $opts, $channel = null) {
        parent::__construct($hostname, $opts, $channel);
    }

    /**
     * @param \FeatureChaos\GetAllFeatureRequest $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     * @return \Grpc\ServerStreamingCall
     */
    public function Subscribe(\FeatureChaos\GetAllFeatureRequest $argument,
      $metadata = [], $options = []) {
        return $this->_serverStreamRequest('/FeatureChaos.FeatureService/Subscribe',
        $argument,
        ['\FeatureChaos\GetFeatureResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param array $metadata metadata
     * @param array $options call options
     * @return \Grpc\ClientStreamingCall
     */
    public function Stats($metadata = [], $options = []) {
        return $this->_clientStreamRequest('/FeatureChaos.FeatureService/Stats',
        ['\Google\Protobuf\GPBEmpty','decode'],
        $metadata, $options);
    }

}
