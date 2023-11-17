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


[English](./README.md) | 简体中文

## 介绍

Koordinator已经支持了K8s生态内的在离线混部，通过Batch超卖资源以及BE QoS，离线任务可以使用到集群内的空闲资源，提升资源使用效率。然而，
在K8s生态外，仍有相当数量的应用运行在其他资源管理系统，例如Apache Hadoop YARN。作为大数据生态下的资源管理系统，YARN承载了包括MapReduce、
Spark、Flink以及Presto等在内的多种计算引擎。

为了进一步丰富Koordinator支持的在离线混部场景，Koordinator社区提供了面向大数据场景的YARN混部套件`Koordinator YARN Copilot`,
用于支持Hadoop YARN应用与K8s混部，将Koordiantor的Batch资源提供给Hadoop YARN使用，进一步提升集群资源的使用效率。
`Koordinator YARN Copilot`具备以下特点：

- 面向开源生态：针对开源版本的Hadoop YARN实现，无需对YARN本身做侵入式改造。
- 统一资源优先级和QoS策略：YARN混部套件完全对标Koordinator的Batch资源模型，同时接受单机一系列QoS策略的管控。
- 节点资源共享：在同一节点上可以同时运行Batch类型的Pod和YARN的Task。
- 适应多种环境：YARN混部套件对集群类型没有约束，可以在包括公共云、IDC等多种场景下使用。

## 快速开始

你可以在 [Koordinator website](https://koordinator.sh/docs) 查看到完整的文档集。

- 安装/升级 Koordinator [最新版本](https://koordinator.sh/docs/installation)
- 参考[最佳实践](https://koordinator.sh/zh-Hans/docs/next/best-practices/colocation-of-hadoop-yarn/)，里面包含了关于K8s与YARN混部的详细示例。

## 行为守则

Koordinator 社区遵照[行为守则](https://github.com/koordinator-sh/koordinator/CODE_OF_CONDUCT.md) 。我们鼓励每个人在参与之前先读一下它。

为了营造一个开放和热情的环境，我们作为贡献者和维护者承诺：无论年龄、体型、残疾、种族、经验水平、教育程度、社会经济地位、国籍、个人外貌、种族、宗教或性认同和性取向如何，参与我们的项目和社区的每个人都不会受到骚扰。

## 贡献

我们非常欢迎每一位社区同学共同参与 Koordinator 的建设，你可以从 [CONTRIBUTING.md](https://github.com/koordinator-sh/koordinator/CONTRIBUTING.md) 手册开始。

## 成员

我们鼓励所有贡献者成为成员。我们的目标是发展一个由贡献者、审阅者和代码所有者组成的活跃、健康的社区。在我们的[社区成员](https://github.com/koordinator-sh/community/blob/main/community-membership.md)页面，详细了解我们的成员要求和责任。

## 社区

在 [koordinator-sh/community 仓库](https://github.com/koordinator-sh/community) 中托管了所有社区信息， 例如成员制度、代码规范等。

我们鼓励所有贡献者成为成员。我们的目标是发展一个由贡献者、审阅者和代码所有者组成的活跃、健康的社区。
请在[社区成员制度](https://github.com/koordinator-sh/community/blob/main/community-membership.md)页面，详细了解我们的成员要求和责任。

活跃的社区途径：

- 社区双周会（中文）：
  - 周二 19:30 GMT+8 (北京时间)
  - [钉钉会议链接](https://meeting.dingtalk.com/j/cgTTojEI8Zy)
  - [议题&记录文档](https://shimo.im/docs/m4kMLdgO1LIma9qD)
- Slack( English ): [koordinator channel](https://kubernetes.slack.com/channels/koordinator) in Kubernetes workspace
- 钉钉( Chinese ): 搜索群ID `33383887`或者扫描二维码加入

<div>
  <img src="https://github.com/koordinator-sh/koordinator/raw/main/docs/images/dingtalk.png" width="300" alt="Dingtalk QRCode">
</div>

## License

Koordinator is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
<!--

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=koordinator-sh/koordinator&type=Date)](https://star-history.com/#koordinator-sh/koordinator&Date)
-->