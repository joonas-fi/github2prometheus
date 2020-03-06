![Build status](https://github.com/joonas-fi/github2prometheus/workflows/Build/badge.svg)
[![Download](https://img.shields.io/github/downloads/joonas-fi/github2prometheus/total.svg?style=for-the-badge)](https://github.com/joonas-fi/github2prometheus/releases)

GitHub repo statistics to Prometheus from AWS Lambda.


How to deploy
-------------

Follow the same instructions as in [Onni](https://github.com/function61/onni).

```
$ version="..."; deployer deploy github2prometheus "https://dl.bintray.com/joonas/dl/github2prometheus/$version/deployerspec.zip"
```
