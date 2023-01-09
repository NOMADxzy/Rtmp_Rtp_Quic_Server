package main

import (
	"container/list"
	"sync"
)

type rtpQueueItem struct {
	packet *RTPPacket
	seq    uint16
}

type queue struct {
	m            sync.RWMutex
	maxSize      int
	bytesInQueue int
	queue        *list.List
}

func newQueue(size int) *queue {
	return &queue{queue: list.New(), maxSize: size}
}

func (q *queue) SizeOfNextRTP() int {
	q.m.RLock()
	defer q.m.RUnlock()

	if q.queue.Len() <= 0 {
		return 0
	}

	return q.queue.Front().Value.(rtpQueueItem).packet.MarshalSize()
}

func (q *queue) SeqNrOfNextRTP() uint16 {
	q.m.RLock()
	defer q.m.RUnlock()

	if q.queue.Len() <= 0 {
		return 0
	}

	return q.queue.Front().Value.(rtpQueueItem).packet.GetSeq()
}

func (q *queue) SeqNrOfLastRTP() uint16 {
	q.m.RLock()
	defer q.m.RUnlock()

	if q.queue.Len() <= 0 {
		return 0
	}

	return q.queue.Back().Value.(rtpQueueItem).packet.GetSeq()
}

func (q *queue) BytesInQueue() int {
	q.m.Lock()
	defer q.m.Unlock()

	return q.bytesInQueue
}

func (q *queue) SizeOfQueue() int {
	q.m.RLock()
	defer q.m.RUnlock()

	return q.queue.Len()
}

func (q *queue) Clear() int {
	q.m.Lock()
	defer q.m.Unlock()

	size := q.queue.Len()
	q.bytesInQueue = 0
	q.queue.Init()
	return size
}

func (q *queue) Enqueue(pkt *RTPPacket, seq uint16) {
	q.m.Lock()
	defer q.m.Unlock()

	q.bytesInQueue += len(pkt.buffer) + len(pkt.ekt)
	q.queue.PushBack(rtpQueueItem{
		packet: pkt,
		seq:    seq,
	})
	if q.queue.Len() > q.maxSize { //超出最大长度
		front := q.queue.Front()
		q.queue.Remove(front)
		q.bytesInQueue -= front.Value.(rtpQueueItem).packet.MarshalSize()
	}
}

func (q *queue) Dequeue() *RTPPacket {
	q.m.Lock()
	defer q.m.Unlock()

	if q.queue.Len() <= 0 {
		return nil
	}

	front := q.queue.Front()
	q.queue.Remove(front)
	packet := front.Value.(rtpQueueItem).packet
	q.bytesInQueue -= packet.MarshalSize()
	return packet
}

func (q *queue) GetPkt(targetSeq uint16) *RTPPacket {
	front := q.queue.Front()
	for cur := front; cur != nil; cur = cur.Next() {
		cur_seq := cur.Value.(rtpQueueItem).seq
		if cur_seq < targetSeq { //还没到
			continue
		} else if cur_seq == targetSeq {
			return cur.Value.(rtpQueueItem).packet
		} else { //不存在了
			return nil
		}
	}
	return nil
}
