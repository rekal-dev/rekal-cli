# Download metrics

The daily Download metrics workflow snapshots public release asset counters and retains JSON artifacts for 90 days. Run it manually from Actions or run `python3 scripts/download-metrics.py` locally.

`archive_downloads` sums only `rekal_*.tar.gz`, excluding checksums and source archives. Downloads are our acquisition success signal, not unique users or verified installs; retries, upgrades and tests count too. Compare assets by ID across snapshots because replacing an asset resets its counter.

The website uses this collector during its scheduled builds to publish `/download-metrics.json` and `/downloads.json`. The footer shows the total and capture date; the README badge uses that same total. Collection failure leaves the previous successful website and date intact. Website clicks remain in GA4.
