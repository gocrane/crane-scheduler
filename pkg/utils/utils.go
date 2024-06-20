package utils

import (
	"net"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	TimeFormat       = "2006-01-02T15:04:05Z"
	DefaultTimeZone  = "Asia/Shanghai"
	DefaultNamespace = "crane-system"
	IPv4             = "IPv4"
	IPv6             = "IPv6"
)

// IsDaemonsetPod judges if this pod belongs to one daemonset workload.
func IsDaemonsetPod(pod *corev1.Pod) bool {
	for _, ownerRef := range pod.GetOwnerReferences() {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

func GetLocalTime() string {
	loc := GetLocation()
	if loc == nil {
		time.Now().Format(TimeFormat)
	}

	return time.Now().In(loc).Format(TimeFormat)
}

func GetLocation() *time.Location {
	zone := os.Getenv("TZ")

	if zone == "" {
		zone = DefaultTimeZone
	}

	loc, _ := time.LoadLocation(zone)

	return loc
}

func GetSystemNamespace() string {
	ns := os.Getenv("CRANE_SYSTEM_NAMESPACE")

	if ns == "" {
		ns = DefaultNamespace
	}

	return ns
}

// NormalizaScore nornalize the score in range [min, max]
func NormalizeScore(value, max, min int64) int64 {
	if value < min {
		value = min
	}

	if value > max {
		value = max
	}

	return value
}

func IsValidIPv4(ip string) bool {
	addr := net.ParseIP(ip)
	return addr != nil && addr.To4() != nil
}

func IsValidIPv6(ip string) bool {
	addr := net.ParseIP(ip)
	return addr != nil && addr.To4() == nil
}
