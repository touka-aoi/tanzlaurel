package domain

import "fmt"

type IdleReason uint8

const (
	IdleNone     IdleReason = 0
	IdleRead     IdleReason = 1 << 0
	IdleWrite    IdleReason = 1 << 1
	IdlePong     IdleReason = 1 << 2
	IdleDisabled IdleReason = 1 << 7 // timeout<=0 のとき
)

func (r IdleReason) Has(x IdleReason) bool { return r&x != 0 }

func (r IdleReason) String() string {
	if r == IdleNone {
		return "none"
	}
	if r == IdleDisabled {
		return "disabled"
	}
	out := ""
	add := func(s string) {
		if out == "" {
			out = s
			return
		}
		out += "|" + s
	}
	if r.Has(IdleRead) {
		add("read")
	}
	if r.Has(IdleWrite) {
		add("write")
	}
	if r.Has(IdlePong) {
		add("pong")
	}
	if out == "" {
		return fmt.Sprintf("unknown(%d)", r)
	}
	return out
}
