# goyarn
use goyarn as native go clients for Apache Hadoop YARN.

It includes an early version of Hadoop IPC client and requisite YARN client libraries to implement YARN applications completely in go (both YARN application-client and application-master).

Koordinator extends `github.com/hortonworks/gohadoop` by implementing resource manager administration service and other clients.

# Notes:
Set HADOOP_CONF_DIR environment variable, and ensure the conf directory contains both *-default.xml and *-site.xml files.
rm_update_node_resource.go is an example go YARN rpc client of rm-admin: call update node resource to do the updates.

# Run rm_update_node_resource
change the `host` and `port` to target node id
$ HADOOP_CONF_DIR=conf go run pkg/yarn/client/examples/rm_update_node_resource.go