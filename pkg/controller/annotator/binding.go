package annotator

import (
	"container/heap"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// Binding is a concise struction of pod binding records,
// which consists of pod name, namespace name, node name and accurate timestamp.
// Note that we only record temporary Binding imformation.
type Binding struct {
	Node      string
	Namespace string
	PodName   string
	Timestamp int64
}

// BindingHeap is a Heap struction storing Binding imfromation.
type BindingHeap []*Binding

func (b BindingHeap) Len() int {
	return len(b)
}

func (b BindingHeap) Less(i, j int) bool {
	return b[i].Timestamp < b[j].Timestamp
}

func (b BindingHeap) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b *BindingHeap) Pop() interface{} {
	old := *b
	n := len(old)
	x := old[n-1]
	*b = old[0 : n-1]
	return x
}

func (b *BindingHeap) Push(x interface{}) {
	*b = append(*b, x.(*Binding))
}

// BindingRecords relizes an Binding heap, which limits heap size,
// and recycles automatically, contributing to store latest and least imformation.
type BindingRecords struct {
	size        int32
	bindings    *BindingHeap
	gcTimeRange time.Duration
	rw          sync.RWMutex
}

// NewBindingRecords returns an BindingRecords object.
func NewBindingRecords(size int32, tr time.Duration) *BindingRecords {
	bh := BindingHeap{}
	heap.Init(&bh)
	return &BindingRecords{
		size:        size,
		bindings:    &bh,
		gcTimeRange: tr,
	}
}

// AddBinding add new Binding to BindingHeap.
func (br *BindingRecords) AddBinding(b *Binding) {
	br.rw.Lock()
	defer br.rw.Unlock()

	if br.bindings.Len() == int(br.size) {
		heap.Pop(br.bindings)
	}

	heap.Push(br.bindings, b)
}

// GetLastNodeBindingCount caculates how many pods scheduled on specified node recently.
func (br *BindingRecords) GetLastNodeBindingCount(node string, timeRange time.Duration) int {
	br.rw.RLock()
	defer br.rw.RUnlock()

	cnt, timeline := 0, time.Now().UTC().Unix()-int64(timeRange.Seconds())

	for _, binding := range *br.bindings {
		if binding.Timestamp > timeline && binding.Node == node {
			cnt++
		}
	}

	klog.V(4).Infof("The total Binding count is %d, while node[%s] count is %d",
		len(*br.bindings), node, cnt)

	return cnt
}

// BindingsGC recycles expired Bindings.
func (br *BindingRecords) BindingsGC() {
	br.rw.Lock()
	defer br.rw.Unlock()

	klog.V(4).Infof("GC period is %f", br.gcTimeRange.Seconds())

	if br.gcTimeRange == 0 {
		return
	}

	timeline := time.Now().UTC().Unix() - int64(br.gcTimeRange.Seconds())
	for br.bindings.Len() > 0 {
		binding := heap.Pop(br.bindings).(*Binding)

		klog.V(6).Infof("Try to recycle Binding(%v) with timeline %d", binding, timeline)

		if binding.Timestamp > timeline {
			heap.Push(br.bindings, binding)
			return
		}
	}

	return
}
