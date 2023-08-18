FROM alpine:3.14 as BUILDER

ENV HADOOP_VERSION 3.3.2
ENV SPARK_VERSION 3.3.2

RUN apk update \
    && apk --update add curl \
    && rm -rf /var/cache/apk/* /tmp/* /var/tmp/* $HOME/.cache
RUN curl -s -o /tmp/hadoop.tgz https://mirrors.aliyun.com/apache/hadoop/common/hadoop-${HADOOP_VERSION}/hadoop-${HADOOP_VERSION}.tar.gz \
    && tar --directory /opt -xzf /tmp/hadoop.tgz
RUN curl -s -o /tmp/spark.tgz https://mirrors.aliyun.com/apache/spark/spark-${SPARK_VERSION}/spark-${SPARK_VERSION}-bin-hadoop3.tgz \
    && tar --directory /opt -xzf /tmp/spark.tgz


FROM openjdk:8

ENV HADOOP_VERSION 3.3.2
ENV SPARK_VERSION 3.3.2
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

WORKDIR $HADOOP_HOME
