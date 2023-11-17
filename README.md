<h1 align="center">
  <p align="center">Koordinator YARN Copilot</p>
  <a href="https://koordinator.sh"><img src="https://github.com/koordinator-sh/koordinator/raw/main/docs/images/koordinator-logo.jpeg" alt="Koordinator"></a>
</h1>

[![License](https://img.shields.io/github/license/koordinator-sh/koordinator.svg?color=4EB1BA&style=flat-square)](https://opensource.org/licenses/Apache-2.0)
[![GitHub release](https://img.shields.io/github/v/release/koordinator-sh/yarn-copilot.svg?style=flat-square)](https://github.com/koordinator-sh/yarn-copilot/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/koordinator-sh/yarn-copilot/ci.yaml?label=CI&logo=github&style=flat-square&branch=main)](https://github.com/koordinator-sh/yarn-copilot/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/koordinator-sh/yarn-copilot?style=flat-square)](https://goreportcard.com/report/github.com/koordinator-sh/yarn-copilot)
[![codecov](https://img.shields.io/codecov/c/github/koordinator-sh/yarn-copilot?logo=codecov&style=flat-square)](https://codecov.io/github/koordinator-sh/yarn-copilot)
[![PRs Welcome](https://badgen.net/badge/PRs/welcome/green?icon=https://api.iconify.design/octicon:git-pull-request.svg?color=white&style=flat-square)](CONTRIBUTING.md)
[![Slack](https://badgen.net/badge/slack/join/4A154B?icon=slack&style=flat-square)](https://join.slack.com/t/koordinator-sh/shared_invite/zt-1756qoub4-Cn4~esfdlfAPsD7cwO2NzA)


English | [简体中文](./README-zh_CN.md)
## Introduction

Koordinator has supported hybrid orchestration workloads on Kubernetes, so that batch jobs can use the requested but unused resource
as `koord-batch` priority and `BE` QoS class to improve the cluster utilization. However, there still lots of applications
running beyond K8s such as Apache Haddop YARN. As a resource management platform in BigData ecosystem, YARN has supported
numbers of computing engines including MapReduce, Spark, Flink, Presto, etc.

In order to extend the co-location scenario of Koordinator, now the community has provided Hadoop YARN extended suits
`Koordinator YARN Copilot` in BigData ecosystem, supporting running Hadoop YARN jobs by `koord-batch` resources with 
other K8s pods. The `Koordinator YARN Copilot` has following characters:

- Open-Source native: implement against open-sourced version of Hadoop YARN; so there is no hack inside YARN modules.
- Unifed resource priority and QoS strategy: the suits aims to the `koord-batch` priority of Koordinator, and also managed by QoS strategies of koordlet.
- Resource sharing on node level: node resources of `koord-batch` priority can be requested by tasks of YARN or `Batch` pods both. 
- Adaptive for multiple environments: the suits can be run under any environment, including public cloud or IDC.

## Quick Start

You can view the full documentation from the [Koordinator website](https://koordinator.sh/docs).

- Install or upgrade Koordinator with [the latest version](https://koordinator.sh/docs/installation).
- Referring to [best practices](https://koordinator.sh/docs/next/best-practices/colocation-of-hadoop-yarn), there will be
  detailed instructions for running Hadoop YARN jobs with Koordinator batch resources in K8s.

## Code of conduct

The Koordinator community is guided by our [Code of Conduct](https://github.com/koordinator-sh/koordinator/CODE_OF_CONDUCT.md),
which we encourage everybody to read before participating.

In the interest of fostering an open and welcoming environment, we as contributors and maintainers pledge to making
participation in our project and our community a harassment-free experience for everyone, regardless of age, body size,
disability, ethnicity, level of experience, education, socio-economic status,
nationality, personal appearance, race, religion, or sexual identity and orientation.

## Contributing

You are warmly welcome to hack on Koordinator. We have prepared a detailed guide [CONTRIBUTING.md](https://github.com/koordinator-sh/koordinator/ONTRIBUTING.md).

## Community

The [koordinator-sh/community repository](https://github.com/koordinator-sh/community) hosts all information about
the community, membership and how to become them, developing inspection, who to contact about what, etc.

We encourage all contributors to become members. We aim to grow an active, healthy community of contributors, reviewers,
and code owners. Learn more about requirements and responsibilities of membership in
the [community membership](https://github.com/koordinator-sh/community/blob/main/community-membership.md) page.

Active communication channels:

- Bi-weekly Community Meeting (APAC, *Chinese*):
  - Tuesday 19:30 GMT+8 (Asia/Shanghai)
  - [Meeting Link(DingTalk)](https://meeting.dingtalk.com/j/cgTTojEI8Zy)
  - [Notes and agenda](https://shimo.im/docs/m4kMLdgO1LIma9qD)
- Slack(English): [koordinator channel](https://kubernetes.slack.com/channels/koordinator) in Kubernetes workspace
- DingTalk(Chinese): Search Group ID `33383887` or scan the following QR Code

<div>
  <img src="https://github.com/koordinator-sh/koordinator/raw/main/docs/images/dingtalk.png" width="300" alt="Dingtalk QRCode">
</div>

## License

Koordinator is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
<!--

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=koordinator-sh/koordinator&type=Date)](https://star-history.com/#koordinator-sh/koordinator&Date)
-->