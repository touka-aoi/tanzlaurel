package application

type InputEntry struct {
	EntityID EntityID
	KeyMask  uint32
}

type PendingInput struct {
	queue []InputEntry
}

func (p *PendingInput) Push(entry InputEntry) {
	p.queue = append(p.queue, entry)
}

func (p *PendingInput) Drain() []InputEntry {
	flushed := p.queue
	p.queue = nil
	return flushed
}
