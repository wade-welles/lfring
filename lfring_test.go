/*
* MIT License
*
* Copyright (c) 2017 Milad (Mike) Taghavi <mitghi[at]me/gmail.com>
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
 */

package lfring

import (
	"fmt"
	"testing"
	"unsafe"
)

// - MARK: Test-structs section.

type tstsample struct {
	value int
}

type tstslice struct {
	nodes []unsafe.Pointer
	sid   string
}

type tststore struct {
	counter uint64
	nodes   *tstslice
}

// - MARK: Tests section.

func TestPointerTagging(t *testing.T) {
	const sval = 8
	var (
		s    *tstsample     = &tstsample{value: sval}
		tptr unsafe.Pointer // tagged pointer
		tag  uint           // tag value
		err  error
	)
	tptr, err = TaggedPointer(unsafe.Pointer(s), 1)
	if err != nil {
		t.Fatal("assertion failed, expected==nil.")
	}
	if (uintptr(tptr) & 0x3) != 0x1 {
		t.Fatal("assertion failed, expected equal.")
	}
	tag = GetTag(tptr)
	if tag != 1 {
		t.Fatal("assertion failed, expected equal.")
	}
	tptr = Untag(unsafe.Pointer(s))
	if uintptr(tptr)&0x3 != 0x0 {
		t.Fatal("assertion failed, expected untagged pointer.")
	}
	s = (*tstsample)(tptr)
	if s == nil || (s.value != sval) {
		t.Fatal("assertion failed, expected s.value==sval.")
	}
	tag = GetTag(tptr)
	if tag != 0 {
		t.Fatal("assertion failed, expected tag==0.")
	}
}

func TestRDCSS(t *testing.T) {
	var (
		cnode  *tststore      // current node
		nnode  *tststore      // new node
		cntptr unsafe.Pointer // counter pointer
	)
	cnode = &tststore{
		counter: 16,
		nodes:   &tstslice{make([]unsafe.Pointer, 0), "original"},
	}
	nnode = &tststore{
		counter: 16,
		nodes:   &tstslice{make([]unsafe.Pointer, 0), "replaced"},
	}
	cntptr = unsafe.Pointer(&cnode.counter)
	// swap underlaying `nodes` iff counters
	// are equal
	if !RDCSS(
		(*unsafe.Pointer)(cntptr),
		(unsafe.Pointer)(unsafe.Pointer(uintptr(nnode.counter))),
		(*unsafe.Pointer)(unsafe.Pointer(&cnode.nodes)),
		unsafe.Pointer(cnode.nodes),
		unsafe.Pointer(nnode.nodes),
	) {
		t.Fatal("Expected RDCSS to succeed")
	}
	if cnode.nodes == nil {
		t.Fatal("inconsistent state.")
	}
	if cnode.nodes.sid != "replaced" {
		t.Fatal("inconsistent state.", cnode.nodes)
	}
}

func TestRDCSS2(t *testing.T) {
	var (
		s       *tstsample     = &tstsample{value: 64}
		n       *tstsample     = &tstsample{value: 128}
		ring    *Ring          = NewRing(8)
		slotptr unsafe.Pointer = unsafe.Pointer(OffsetSliceSlot(unsafe.Pointer(&ring.nodes), 1, archPTRSIZE))
		rdiPtr  unsafe.Pointer = unsafe.Pointer(&ring.rdi)
		vptr    **tstsample    // value pointer
	)
	if !SetSliceSlot(unsafe.Pointer(&ring.nodes), 1, archPTRSIZE, unsafe.Pointer(&s)) {
		t.Fatal("inconsistent state, can't write to slice/slot.")
	}
	if ring.nodes == nil {
		t.Fatal("assertion failed, fatal condition from incorrent pointer mutation.")
	}
	if ring.nodes[1] == nil {
		t.Fatal("assertion failed, expected non-nil.")
	}
	vptr = (**tstsample)(ring.nodes[1])
	if vptr == nil {
		t.Fatal("assertion failed, expected non-nil.")
	} else if *vptr == nil {
		t.Fatal("assertion failed, expected non-nil.")
	}
	if (*vptr) != s {
		t.Fatal("assertion failed, expected equal as 's' is written to slot 1.")
	}
	if (((*vptr).value != s.value) && s.value == 64) || s.value != 64 {
		t.Fatal("assertion failed, invalid pointers.")
	}
	fmt.Printf("nodes: %#x\n", ring.nodes)
	if !RDCSS(
		(*unsafe.Pointer)(unsafe.Pointer(&rdiPtr)),
		(unsafe.Pointer)(unsafe.Pointer(rdiPtr)),
		(*unsafe.Pointer)(unsafe.Pointer(slotptr)),
		(unsafe.Pointer)(unsafe.Pointer(&s)),
		(unsafe.Pointer)(unsafe.Pointer(n)),
	) {
		t.Fatal("inconsistent state, RDCSS failure.")
	}
	if s == nil {
		t.Fatal("assertion failed, fatal condition from incorrect pointer mutation.")
	} else if s.value != 64 {
		t.Fatal("assertion failed, invalid pointer.")
	}
	fmt.Printf("nodes: %#x\n", ring.nodes)
	if ring.nodes[1] == nil {
		t.Fatal("assertion failed, expected non-nil for occupied slot.")
	}
	if s == nil {
		t.Fatal("assertion failed, fatal condition from incorrect pointer mutation.")
	}
	vptr = (**tstsample)(slotptr)
	if vptr == nil {
		t.Fatal("assertion failed, expected non-nil pointer.")
	} else if *vptr == nil {
		t.Fatal("assertion failed, expected non-nil pointer.")
	}
	if (*vptr) != n {
		t.Fatal("assertion failed, expected equal as 's' is swapped with 'n'.")
	}
	if (((*vptr).value != n.value) && n.value == 128) || n.value != 128 {
		t.Fatal("assertion failed, invalid pointers.")
	}
	fmt.Println("slotptr:", *(**tstsample)(slotptr), "addr:", slotptr, &n, ring.nodes, ring.nodes[1])
}
