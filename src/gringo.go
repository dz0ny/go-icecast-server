// Copyright 2012 Darren Elwood <darren@textnode.com> http://www.textnode.com @textnode
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at 
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Version 0.1
//
// A minimalist queue implemented using a stripped-down lock-free ringbuffer. 
// Inspired by a talk by @mjpt777 at @devbash titled "Lessons in Clean Fast Code" which, 
// among other things, described the LMAX Disruptor.
//
// N.B. To see the performance benefits of gringo versus Go's channels, you must have multiple goroutines
// and GOMAXPROCS > 1.

// Known Limitations:
//
// *) At most (2^64)-2 items can be written to the queue.
// *) The size of the queue must be a power of 2.
// 
// Suggestions:
//
// *) If you have enough cores you can change from runtime.Gosched() to a busy loop.
// 

package gringo

import "sync/atomic"

//import "runtime"

// Example item which we will be writing to and reading from the queue
type Payload struct {
	value []byte
}

func NewPayload(value []byte) *Payload {
	return &Payload{value: value}
}

func (self *Payload) Value() []byte {
	return self.value
}

// The queue
const queueSize uint32 = 8
const indexMask uint32 = queueSize - 1

type Gringo struct {
	lastCommittedIndex uint32
	nextFreeIndex      uint32
	readerIndex        uint32
	contents           [queueSize]Payload
}

func NewGringo() *Gringo {
	return &Gringo{lastCommittedIndex: 0, nextFreeIndex: 1, readerIndex: 1}
}

func (self *Gringo) Write(value Payload) {
	var myIndex = atomic.AddUint32(&self.nextFreeIndex, 1) - 1
	//Wait for reader to catch up, so we don't clobber a slot which it is (or will be) reading
	for myIndex > (self.readerIndex + queueSize - 2) {
		//runtime.Gosched()
	}
	//Write the item into it's slot
	self.contents[myIndex&indexMask] = value
	//Increment the lastCommittedIndex so the item is available for reading
	for !atomic.CompareAndSwapUint32(&self.lastCommittedIndex, myIndex-1, myIndex) {
		//runtime.Gosched()
	}
}

func (self *Gringo) Read() Payload {
	var myIndex = atomic.AddUint32(&self.readerIndex, 1) - 1
	//If reader has out-run writer, wait for a value to be committed
	for myIndex > self.lastCommittedIndex {
		//runtime.Gosched()
	}
	return self.contents[myIndex&indexMask]
}
