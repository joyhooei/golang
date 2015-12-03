package topn

import "container/heap"

type Less func(small interface{}, big interface{}) bool

type GenHeap struct {
	Data []interface{}
	less Less
}

func (h GenHeap) Len() int           { return len(h.Data) }
func (h GenHeap) Less(i, j int) bool { return h.less(h.Data[i], h.Data[j]) }
func (h GenHeap) Swap(i, j int)      { h.Data[i], h.Data[j] = h.Data[j], h.Data[i] }

func (h *GenHeap) Push(x interface{}) {
	h.Data = append(h.Data, x)
}

func (h *GenHeap) Pop() interface{} {
	old := h.Data
	n := len(old)
	x := old[n-1]
	h.Data = old[0 : n-1]
	return x
}

//找出最大的TopN个值
func TopN(candidates []interface{}, less Less, n int) []interface{} {
	realLen := len(candidates)
	if realLen > n {
		realLen = n
	} else {
		//不需要筛选，总数还不超过n个
		return candidates
	}
	h := &GenHeap{make([]interface{}, realLen, realLen+1), less}
	copy(h.Data, candidates[0:realLen])
	heap.Init(h)
	for i := realLen; i < len(candidates); i++ {
		heap.Push(h, candidates[i])
		heap.Pop(h)
	}
	return h.Data
}
