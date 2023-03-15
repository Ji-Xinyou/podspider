package resource

type ResourceNotebook struct{}

type PodMetrics struct {
	Cpu  CpuMetric
	Mem  MemoryMetric
	Net  NetworkMetric
	Disk DiskMetric
}

// the cpu usage matric
type CpuMetric struct {
	cputime_nanosec int64
}

// the memory usage matric
type MemoryMetric struct {
	rss   int64
	cache int64
	swap  int64
}

// the network usage matric
type NetworkMetric struct {
	in_bytes  int64
	out_bytes int64
}

// the disk usage matric
type DiskMetric struct {
	r int64
	w int64
}
