package application

type PendingInput struct {
	queue [][]byte
}

func (p *PendingInput) Push(input []byte) {
	// コピーして所有権を確保
	copied := make([]byte, len(input))
	copy(copied, input)
	p.queue = append(p.queue, copied)
}

func (p *PendingInput) Drain() [][]byte {
	flushed := p.queue
	p.queue = nil
	return flushed
}
