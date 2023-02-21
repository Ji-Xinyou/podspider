package resource

import (
	"context"
	"flag"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

// ========== SOME HELPERS BELOW
// return the clientset of the cluster, this works as a handle
// this reads config from ~/.kube/config of the machine where this program is running
func ClusterHandle(kubeconfig string) *kubernetes.Clientset {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

func MetricHandle(kubeconfig string) *versioned.Clientset {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	metricsClientset, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return metricsClientset
}

func GenerateKubeconfig() string {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	return *kubeconfig
}

// HELPERS ABOVE

type ResourceManager struct {
	// the path where the kubeconfig is
	kubeconfig string
	// the handle to get the cluster info
	cluster_handle *kubernetes.Clientset
	// the handle to get the cluster metrics
	metric_handle *versioned.Clientset
	// the namespace this resource manager is watching
	watched_ns string
	// pod name -> container name -> term -> value
	resources map[string]map[string]map[string]int64
}

// Initialization
func (mgr *ResourceManager) Start(ns string) {
	mgr.kubeconfig = GenerateKubeconfig()
	mgr.cluster_handle = ClusterHandle(mgr.kubeconfig)
	mgr.metric_handle = MetricHandle(mgr.kubeconfig)

	mgr.watched_ns = ns
	mgr.resources = make(map[string]map[string]map[string]int64)
}

func (mgr *ResourceManager) getPods() *v1.PodList {
	// lists the nodes' info of the cluster
	pods, err := mgr.cluster_handle.CoreV1().Pods(mgr.watched_ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	return pods
}

// return the metrics of the pod, given the pod name and namespace
func (mgr *ResourceManager) getPodMetrics(pod v1.Pod) *v1beta1.PodMetrics {
	metrics, err := mgr.metric_handle.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})

	// Some of the time, the metric server cannot return the metrics of the pod
	// Therefore this should never panic
	if err != nil {
		log.Debug(err.Error())
	}

	return metrics
}

// RecordPodMetrics records the pod metrics
// including cpu usage/request/limit, memory usage/request/limit
//
// the cpu usage is in millicore
// the memory usage is in MiB
func (mgr *ResourceManager) recordPodMetrics() {
	pods := mgr.getPods()

	// pod name -> container name ->
	for _, pod := range pods.Items {
		if pod.Status.Phase != v1.PodRunning {
			continue
		}

		metrics := mgr.getPodMetrics(pod)

		// records usage
		for _, container := range metrics.Containers {
			if mgr.resources[pod.Name] == nil {
				mgr.resources[pod.Name] = make(map[string]map[string]int64)
				mgr.resources[pod.Name][container.Name] = make(map[string]int64)
			}

			mgr.resources[pod.Name][container.Name]["cpu_usage"] = container.Usage.Cpu().MilliValue()
			mgr.resources[pod.Name][container.Name]["mem_usage"] = container.Usage.Memory().MilliValue() / 1e9
		}

		// record container spec
		for _, container := range pod.Spec.Containers {
			if mgr.resources[pod.Name] == nil {
				mgr.resources[pod.Name] = make(map[string]map[string]int64)
				mgr.resources[pod.Name][container.Name] = make(map[string]int64)
			}

			resources := container.Resources
			mgr.resources[pod.Name][container.Name]["cpu_limit"] = resources.Limits.Cpu().MilliValue()
			mgr.resources[pod.Name][container.Name]["mem_limit"] = resources.Limits.Memory().MilliValue() / 1e9
			mgr.resources[pod.Name][container.Name]["cpu_request"] = resources.Requests.Cpu().MilliValue()
			mgr.resources[pod.Name][container.Name]["mem_request"] = resources.Requests.Memory().MilliValue() / 1e9
		}
	}
}

// ========== GETTERS ==========
func (mgr *ResourceManager) GetCpuRequest(podname string, ctrname string) int64 {
	return mgr.resources[podname][ctrname]["cpu_request"]
}

func (mgr *ResourceManager) GetCpuLimit(podname string, ctrname string) int64 {
	return mgr.resources[podname][ctrname]["cpu_limit"]
}

func (mgr *ResourceManager) GetCpuUsage(podname string, ctrname string) int64 {
	return mgr.resources[podname][ctrname]["cpu_usage"]
}

func (mgr *ResourceManager) GetMemRequest(podname string, ctrname string) int64 {
	return mgr.resources[podname][ctrname]["mem_request"]
}

func (mgr *ResourceManager) GetMemLimit(podname string, ctrname string) int64 {
	return mgr.resources[podname][ctrname]["mem_limit"]
}

func (mgr *ResourceManager) GetMemUsage(podname string, ctrname string) int64 {
	return mgr.resources[podname][ctrname]["mem_usage"]
}

// ========== DUMPERS ==========
func (mgr *ResourceManager) DumpPods() {
	pods := mgr.getPods()
	npod := len(pods.Items)

	for _, pod := range pods.Items {
		name, namespace := pod.Name, pod.Namespace
		log.WithFields(log.Fields{
			"npod":          npod,
			"pod name":      name,
			"pod namespace": namespace,
		}).Info("Dumping pods")
	}
}

// if namespace is empty, then dump all pods
func (mgr *ResourceManager) DumpPodMetrics(namespace string) {
	pods := mgr.getPods()

	for _, pod := range pods.Items {
		dump := namespace == "" || namespace == pod.Namespace
		if dump && pod.Status.Phase == v1.PodRunning {
			mgr.showStats(pod)
		}
	}
}

// this dumps some messages, the function is stupid since
// there maybe multiple containers in a pod
func (mgr *ResourceManager) showStats(pod v1.Pod) {
	m := mgr.resources[pod.Name]
	ctr := pod.Spec.Containers[0]

	log.WithFields(log.Fields{
		"podname":         pod.Name,
		"node":            pod.Spec.NodeName,
		"cpu usage(mili)": m[ctr.Name]["cpu_usage"],
		"mem usage(MiB)":  m[ctr.Name]["mem_usage"],
		"cpu req(mili)":   m[ctr.Name]["cpu_request"],
		"cpu limit(mili)": m[ctr.Name]["cpu_limit"],
		"mem req(MiB)":    m[ctr.Name]["mem_request"],
		"mem limit(MiB)":  m[ctr.Name]["mem_limit"],
	}).Info("Dumping Pod usage")
}

func (mgr *ResourceManager) Tick() {
	mgr.recordPodMetrics()
	mgr.DumpPodMetrics(mgr.watched_ns)
}
