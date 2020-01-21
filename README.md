[![Build Status](https://img.shields.io/travis/joonas-fi/github2prometheus.svg?style=for-the-badge)](https://travis-ci.org/joonas-fi/github2prometheus)
[![Download](https://img.shields.io/badge/Download-bintray%20latest-blue.svg?style=for-the-badge)](https://bintray.com/joonas/dl/github2prometheus/_latestVersion#files)

Push GitHub repo statistics to Prometheus from AWS Lambda.

NOTE: currently we're using [prompipe](https://github.com/function61/prompipe) to push
the data, but ideally we'd use Prometheus' pull model.. it's just that the endpoint shouldn't
be polled every 5s to stay within usage quotas.. and our Prometheus autodiscovery doesn't
yet support modifying scrape intervals .. and Prometheus's push gateway doesn't seem to
support explicit timestamps.


How to deploy
-------------

Follow the same instructions as in [Onni](https://github.com/function61/onni).

```
$ version="..."; deployer deploy github2prometheus "https://dl.bintray.com/joonas/dl/github2prometheus/$version/deployerspec.zip"
```
