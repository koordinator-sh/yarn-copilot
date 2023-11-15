FROM alpine:3.14 as BUILDER

ENV HADOOP_VERSION 3.3.3
ENV SPARK_VERSION 3.3.3

RUN apk update \
    && apk --update add curl \
    && rm -rf /var/cache/apk/* /tmp/* /var/tmp/* $HOME/.cache
RUN curl -s -o /tmp/hadoop.tgz https://mirrors.aliyun.com/apache/hadoop/common/hadoop-${HADOOP_VERSION}/hadoop-${HADOOP_VERSION}.tar.gz \
    && tar --directory /opt -xzf /tmp/hadoop.tgz
RUN curl -s -o /tmp/spark.tgz https://mirrors.aliyun.com/apache/spark/spark-${SPARK_VERSION}/spark-${SPARK_VERSION}-bin-hadoop3.tgz \
    && tar --directory /opt -xzf /tmp/spark.tgz


FROM openjdk:8

ENV HADOOP_VERSION 3.3.3
ENV SPARK_VERSION 3.3.3
ENV SPARK_HOME=/opt/spark
ENV HADOOP_HOME=/opt/hadoop

ENV HADOOP_COMMON_HOME=${HADOOP_HOME} \
    HADOOP_HDFS_HOME=${HADOOP_HOME} \
    HADOOP_MAPRED_HOME=${HADOOP_HOME} \
    HADOOP_YARN_HOME=${HADOOP_HOME} \
    HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop \
    PATH=${PATH}:${HADOOP_HOME}/bin:${SPARK_HOME}/bin

COPY --from=BUILDER /opt/hadoop-${HADOOP_VERSION} ${HADOOP_HOME}
COPY --from=BUILDER /opt/spark-${SPARK_VERSION}-bin-hadoop3 ${SPARK_HOME}

RUN apt-get update && apt-get install -y apt-transport-https
RUN curl https://mirrors.aliyun.com/kubernetes/apt/doc/apt-key.gpg | apt-key add -
RUN echo 'deb https://mirrors.aliyun.com/kubernetes/apt/ kubernetes-xenial main' >> /etc/apt/sources.list.d/kubernetes.list
#RUN cat <<EOF >/etc/apt/sources.list.d/kubernetes.list \
#    deb https://mirrors.aliyun.com/kubernetes/apt/ kubernetes-xenial main \
#    EOF

RUN apt-get update
RUN apt-get install -y kubectl dnsutils

WORKDIR $HADOOP_HOME