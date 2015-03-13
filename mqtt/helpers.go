package mqtt

import "sync/atomic"

type Atomic struct {
	flag uint32
}

func (self *Atomic) Lock() bool {
	return atomic.CompareAndSwapUint32(&self.flag, 0, 1)
}

func (self *Atomic) UnLock() {
	atomic.SwapUint32(&self.flag, 0)
}

func (self *Atomic) isLocked() bool {
	return self.flag > 0
}
