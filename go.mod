module github.com/koordinator-sh/goyarn

go 1.17

require (
	github.com/koordinator-sh/koordinator v1.2.0
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/spf13/pflag v1.0.5
	google.golang.org/protobuf v1.30.0
	k8s.io/api v0.27.2
	k8s.io/apimachinery v0.27.2
	k8s.io/client-go v0.27.2
	k8s.io/cri-api v0.22.6
	k8s.io/klog/v2 v2.90.1
	k8s.io/utils v0.0.0-20230209194617-a36077c30491
	sigs.k8s.io/controller-runtime v0.15.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.15.1 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/oauth2 v0.5.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/term v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220502173005-c8bf987b8c21 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.27.2 // indirect
	k8s.io/apiserver v0.26.0 // indirect
	k8s.io/component-base v0.27.2 // indirect
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f // indirect
	sigs.k8s.io/scheduler-plugins v0.22.6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/api => k8s.io/api v0.22.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.6
	k8s.io/apiserver => k8s.io/apiserver v0.22.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.6
	k8s.io/client-go => k8s.io/client-go v0.22.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.6
	k8s.io/code-generator => k8s.io/code-generator v0.22.6
	k8s.io/component-base => k8s.io/component-base v0.22.6
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.6
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.6
	k8s.io/cri-api => k8s.io/cri-api v0.22.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.6
	k8s.io/gengo => k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.10.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.6
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.6
	k8s.io/kubectl => k8s.io/kubectl v0.22.6
	k8s.io/kubelet => k8s.io/kubelet v0.22.6
	k8s.io/kubernetes => k8s.io/kubernetes v1.22.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.6
	k8s.io/metrics => k8s.io/metrics v0.22.6
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.6
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.6
	sigs.k8s.io/descheduler => sigs.k8s.io/descheduler v0.26.1-0.20230216092500-02b1e8b8b9c1
	sigs.k8s.io/scheduler-plugins => sigs.k8s.io/scheduler-plugins v0.22.6
)
