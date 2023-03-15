package resource

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

// bytes per sector
const sectorSize = 512

// ========== SOME HELPERS BELOW
func (mgr *ResourceManager) generateClientsets() {
	// create the clientset
	clientset, err := kubernetes.NewForConfig(mgr.kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	metricsClientset, err := versioned.NewForConfig(mgr.kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	mgr.clientset = clientset
	mgr.metric_clientset = metricsClientset
}

func generateKubeconfig() *rest.Config {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	return config
}

// HELPERS ABOVE

type ResourceManager struct {
	// the path where the kubeconfig is
	kubeconfig *rest.Config
	// the handle to get the cluster info
	clientset *kubernetes.Clientset
	// the handle to get the cluster metrics
	metric_clientset *versioned.Clientset
	// the namespace this resource manager is watching
	watched_ns string
	// pod name -> container name -> term -> value
	resources map[string]map[string]int64
	// pods name -> metrics
	usage map[string]PodMetrics
}

// Initialization
func (mgr *ResourceManager) Start(ns string) {
	mgr.kubeconfig = generateKubeconfig()
	mgr.generateClientsets()

	mgr.watched_ns = ns
	mgr.resources = make(map[string]map[string]int64)
	mgr.usage = make(map[string]PodMetrics)
}

// also helpers
func (mgr *ResourceManager) getPods() *v1.PodList {
	// lists the nodes' info of the cluster
	pods, err := mgr.clientset.CoreV1().Pods(mgr.watched_ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	return pods
}

// return the metrics of the pod, given the pod name and namespace
func (mgr *ResourceManager) getPodMetrics(pod v1.Pod) *v1beta1.PodMetrics {
	metrics, err := mgr.metric_clientset.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})

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
				mgr.resources[pod.Name] = make(map[string]int64)
			}

			mgr.resources[pod.Name]["cpu_usage"] = container.Usage.Cpu().MilliValue()
			mgr.resources[pod.Name]["mem_usage"] = container.Usage.Memory().MilliValue() / 1e9
		}

		// record container spec
		for _, container := range pod.Spec.Containers {
			if mgr.resources[pod.Name] == nil {
				mgr.resources[pod.Name] = make(map[string]int64)
			}

			resources := container.Resources
			mgr.resources[pod.Name]["cpu_limit"] = resources.Limits.Cpu().MilliValue()
			mgr.resources[pod.Name]["mem_limit"] = resources.Limits.Memory().MilliValue() / 1e9
			mgr.resources[pod.Name]["cpu_request"] = resources.Requests.Cpu().MilliValue()
			mgr.resources[pod.Name]["mem_request"] = resources.Requests.Memory().MilliValue() / 1e9
		}
	}
}

// ! ASSUMES ONE CONTAINER PER POD
// post command "cmd argv" to the pod's container, return its output
// e.g. running "ls /" on the pod's container
func (mgr *ResourceManager) postCommand(pod v1.Pod, cmd string, argv string) (string, error) {
	req := mgr.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", pod.Spec.Containers[0].Name).
		Param("stdout", "true").
		Param("stderr", "true").
		Param("command", cmd).
		Param("command", argv)

	exec, err := remotecommand.NewSPDYExecutor(mgr.kubeconfig, "POST", req.URL())
	if err != nil {
		panic(err)
	}

	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: out,
		Stderr: errOut,
	})

	if err != nil {
		return errOut.String(), err
	}

	return out.String(), nil
}

// ========== GETTERS ==========
func (mgr *ResourceManager) GetCpuRequest(podname string) int64 {
	return mgr.resources[podname]["cpu_request"]
}

func (mgr *ResourceManager) GetCpuLimit(podname string) int64 {
	return mgr.resources[podname]["cpu_limit"]
}

func (mgr *ResourceManager) GetCpuUsage(podname string) int64 {
	return mgr.resources[podname]["cpu_usage"]
}

func (mgr *ResourceManager) GetMemRequest(podname string) int64 {
	return mgr.resources[podname]["mem_request"]
}

func (mgr *ResourceManager) GetMemLimit(podname string) int64 {
	return mgr.resources[podname]["mem_limit"]
}

func (mgr *ResourceManager) GetMemUsage(podname string) int64 {
	return mgr.resources[podname]["mem_usage"]
}

// ========== DUMPERS ==========
func (mgr *ResourceManager) DumpNodes() {
	nodes := mgr.clientset.CoreV1().Nodes()
	nodeList, err := nodes.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	for _, node := range nodeList.Items {
		nodeName := node.Name
		nodeMetrics, err := mgr.metric_clientset.MetricsV1beta1().NodeMetricses().Get(context.Background(), nodeName, metav1.GetOptions{})
		if err != nil {
			log.Debug("Error getting node metrics: ", err)
		}

		cpuUsage := nodeMetrics.Usage.Cpu().MilliValue()
		memoryUsage := nodeMetrics.Usage.Memory().Value()
		cpuCapacity := node.Status.Capacity.Cpu().MilliValue()
		memoryCapacity := node.Status.Capacity.Memory().Value()

		cpu_usage := fmt.Sprintf("%.*f", 4, float64(cpuUsage)/float64(cpuCapacity)*100) + "%"
		mem_usage := fmt.Sprintf("%.*f", 4, float64(memoryUsage)/float64(memoryCapacity)*100) + "%"

		log.Info("Node: ", nodeName, "\tCPU usage: ", cpu_usage, "\tMemory usage: ", mem_usage)
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

	log.WithFields(log.Fields{
		"podname":         pod.Name,
		"node":            pod.Spec.NodeName,
		"cpu usage(mili)": m["cpu_usage"],
		"mem usage(MiB)":  m["mem_usage"],
		"cpu req(mili)":   m["cpu_request"],
		"cpu limit(mili)": m["cpu_limit"],
		"mem req(MiB)":    m["mem_request"],
		"mem limit(MiB)":  m["mem_limit"],
	}).Info("Dumping Pod usage")
}

func (mgr *ResourceManager) genericCat(pod v1.Pod, path string) (string, error) {
	if pod.Status.Phase != v1.PodRunning {
		return "", errors.New("pod is not running")
	}

	out, err := mgr.postCommand(pod, "cat", path)
	if err != nil {
		return "", err
	}

	return out, nil
}

func (mgr *ResourceManager) getCPUMetricForPod(pod v1.Pod) {
	out, err := mgr.genericCat(pod, "/sys/fs/cgroup/cpuacct/cpuacct.usage")
	if err != nil {
		return
	}

	out = strings.TrimSpace(out)

	nano, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		panic(err.Error())
	}

	if metrics, ok := mgr.usage[pod.Name]; ok {
		metrics.Cpu.cputime_nanosec = nano
	}

	log.Info("CPUStat of ", pod.Name, "|cputime_nano: ", nano)

}

func (mgr *ResourceManager) getMemoryMetricForPod(pod v1.Pod) {
	out, err := mgr.genericCat(pod, "/sys/fs/cgroup/memory/memory.stat")
	if err != nil {
		return
	}

	rss, cache, swap := int64(0), int64(0), int64(0)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		kv := strings.Split(line, " ")
		if len(kv) != 2 {
			continue
		}
		k, v := kv[0], kv[1]

		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			panic(err.Error())
		}

		if k == "total_rss" {
			rss = val
		} else if k == "total_cache" {
			cache = val
		} else if k == "total_swap" {
			swap = val
		}
	}

	if metrics, ok := mgr.usage[pod.Name]; ok {
		metrics.Mem.rss = rss
		metrics.Mem.cache = cache
		metrics.Mem.swap = swap
	}

	log.Info("MemoryStat of ", pod.Name, "|rss: ", rss, "|cache: ", cache, "|swap: ", swap)

}

func (mgr *ResourceManager) getNetworkMetricForPod(pod v1.Pod) {
	out, err := mgr.genericCat(pod, "/proc/net/netstat")
	if err != nil {
		return
	}

	inb, outb := int64(0), int64(0)
	lines := strings.Split(out, "\n")

	j := 0
	inIndex, outIndex := 0, 0
	for i, line := range lines {
		headers := strings.Split(line, " ")
		if headers[0] == "IpExt:" {
			j = i + 1

			for k, header := range headers {
				if header == "InOctets" {
					inIndex = k
				} else if header == "OutOctets" {
					outIndex = k
				}
			}
			break
		} else {
			continue
		}
	}

	content := strings.Split(lines[j], " ")

	inb, err = strconv.ParseInt(content[inIndex], 10, 64)
	if err != nil {
		panic(err.Error())
	}

	outb, err = strconv.ParseInt(content[outIndex], 10, 64)

	if err != nil {
		panic(err.Error())
	}

	if metrics, ok := mgr.usage[pod.Name]; ok {
		metrics.Net.in_bytes = inb
		metrics.Net.out_bytes = outb
	}

	log.Info("NetworkStat of ", pod.Name, "|in: ", inb, "|out: ", outb)
}

func (mgr *ResourceManager) getDiskMetricForPod(pod v1.Pod) {
	out, err := mgr.genericCat(pod, "/proc/diskstats")
	if err != nil {
		return
	}

	r, w := int64(0), int64(0)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		kv := strings.Fields(line)
		devname := kv[2]

		// we only care about vda
		if devname != "vda" {
			continue
		}

		r, err = strconv.ParseInt(kv[5], 10, 64)
		if err != nil {
			panic(err.Error())
		}

		w, err = strconv.ParseInt(kv[9], 10, 64)
		if err != nil {
			panic(err.Error())
		}

		break
	}

	w = w * sectorSize
	r = r * sectorSize

	if metrics, ok := mgr.usage[pod.Name]; ok {
		metrics.Disk.w = w
		metrics.Disk.r = r
	}

	log.Info("DiskStat of ", pod.Name, "|r(bytes): ", r, "|w(bytes): ", w)
}

// get the cpu usage in nanosec
func (mgr *ResourceManager) GetCpuMetrics() {
	for _, pod := range mgr.getPods().Items {
		mgr.getCPUMetricForPod(pod)
	}
}

// get the memory rss, swap and cache in byte
func (mgr *ResourceManager) GetMemoryMetrics() {
	for _, pod := range mgr.getPods().Items {
		mgr.getMemoryMetricForPod(pod)
	}
}

// get the network bytes out and in (InOctcts and OutOctcts)
func (mgr *ResourceManager) GetNetworkMetrics() {
	for _, pod := range mgr.getPods().Items {
		mgr.getNetworkMetricForPod(pod)
	}
}

// get the bytes read and written (6th and 10th in diskstats of vda)
func (mgr *ResourceManager) GetDiskMetrics() {
	for _, pod := range mgr.getPods().Items {
		mgr.getDiskMetricForPod(pod)
	}
}

func (mgr *ResourceManager) Tick() {
	mgr.recordPodMetrics()
	// mgr.DumpNodes()
	// mgr.DumpPodMetrics(mgr.watched_ns)

	// print and gather all pod runtime metrics
	mgr.GetCpuMetrics()
	mgr.GetMemoryMetrics()
	mgr.GetNetworkMetrics()
	mgr.GetDiskMetrics()
}
