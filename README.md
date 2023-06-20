# goyarn
use goyarn as native go clients for Apache Hadoop YARN.

It includes an early version of Hadoop IPC client and requisite YARN client libraries to implement YARN applications completely in go (both YARN application-client and application-master).

Koordinator extends `github.com/hortonworks/gohadoop` by implementing resource manager administration service and other clients.

# Notes:
Set HADOOP_CONF_DIR environment variable, and ensure the conf directory contains both *-default.xml and *-site.xml files.
rm_update_node_resource.go is an example go YARN rpc client of rm-admin: call update node resource to do the updates.

# Run rm_update_node_resource example
Change the `host` and `port` to target node id.

Execute command:

```shell script
$ HADOOP_CONF_DIR=conf go run pkg/yarn/client/examples/rm_update_node_resource.go
```

# Run yarn-operator
Install `koordinator` according to [doc](https://koordinator.sh/docs/installation/).

Add annotation to node with YARN node ID
```shell script
kubectl annotate node --overwrite ${k8s.node.name} node.yarn.koordinator.sh=${yarn.node.id}
```

Change the `yarn.resourcemanager.admin.address` in config/manager/configmap.yaml

Execute command:
```shell script
$ kubectl apply -f config/manager/
```