# FeatureChaos Python SDK

Install (local dev):

Run pip install -e . inside sdk/python

Usage:

```python
from featurechaos import FeatureChaosClient, Options

cli = FeatureChaosClient("127.0.0.1:9090", "checkout", Options(auto_send_stats=True))

enabled = cli.is_enabled("new_ui", seed="user:42", attrs={"country": "DE", "tier": "gold"})

cli.close()
```
