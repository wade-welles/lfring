/*
* MIT License
*
* Copyright (c) 2017 Mike Taghavi <mitghi[at]me.com>
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
	"log"
	"sync/atomic"
	"unsafe"
)

// - MARK: Atomics section.

// CASSliceSlot is a function that performs a CAS operation
// on a given slice slot by performing pointer arithmitic
//  to find slot address. `addr` is a pointer to slice,
// `data` is a pointer to old value to be compared,
// `target` is a pointer to the new value,  `index` is
// the slot number and `ptrsize` is the slice value size.
// It returns true when succesfull.
func CASSliceSlot(addr unsafe.Pointer, data unsafe.Pointer, target unsafe.Pointer, index int, ptrsize uintptr) bool {
	var (
		tptr *unsafe.Pointer
		cptr unsafe.Pointer
	)
	tptr = (*unsafe.Pointer)(unsafe.Pointer(*(*uintptr)(addr) + (ptrsize * uintptr(index))))
	cptr = unsafe.Pointer(tptr)
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(cptr)),
		(unsafe.Pointer)(unsafe.Pointer(target)),
		(unsafe.Pointer)(unsafe.Pointer(data)),
	)
}

// OffsetSliceSlot takes a slice pointer and returns
// slot address by adding `index` times `ptrsize` bytes
// to slice data pointer.
func OffsetSliceSlot(addr unsafe.Pointer, index int, ptrsize uintptr) unsafe.Pointer {
	return unsafe.Pointer(*(*uintptr)(addr) + (ptrsize * uintptr(index)))
}

// SetSliceSlot is a wrapper function that writes `d`
// to the given slice slot iff its nil and returns
// true when succesfull.
func SetSliceSlot(addr unsafe.Pointer, index int, ptrsize uintptr, d unsafe.Pointer) bool {
	return CASSliceSlot(addr, d, nil, index, ptrsize)
}

// - MARK: Pointer-Tagging section.

// GetTag returns the tag value from
// low-order bits.
func GetTag(ptr unsafe.Pointer) uint {
	return uint(uintptr(ptr) & uintptr(archMAXTAG))
}

// TaggedPointer is a function for tagging pointers.
// It attaches `tag` value to the pointer `ptr` iff
// `tag` <= `archMAXTAG` and returns the tagged pointer
// along with error set to `nil`. It panics when
// `tag` > `archMAXTAG`, I do too! It's like getting
// headshot by a champagne cork.
func TaggedPointer(ptr unsafe.Pointer, tag uint) (unsafe.Pointer, error) {
	if tag > archMAXTAG {
		// flip the table, not this time!
		panic(EPTRINVALT)
	}
	return unsafe.Pointer(uintptr(ptr) | uintptr(tag)), nil
}

// Untag is a function for untagging pointers. It
// returns a `unsafe.Pointer` with low-order bits
// set to 0.
func Untag(ptr unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(uintptr(ptr) & archPTRMASK)
}

// HasTag returns whether the given pointer `ptr`
// is tagged.
func HasTag(ptr unsafe.Pointer) bool {
	return GetTag(ptr)&archMAXTAG > 0
}

// - MARK: Multi-Word Compare-and-Swap Operation section.

// rdcssDescriptor is descriptor for Multi-Word CAS. RDCSS
// is defined as a restricted form of CAS2 operating atomi-
// cally as follow:
//
// word_t RDCSS(word_t *a1,
//              word_t o1,
//              word_t *a2,
//              word_t o2,
//              word_t n) {
//   r = *a2;
//   if ((r  == o2) && (*a1 == o1)) *a2 = n;
//   return r;
// }
type rdcssDescriptor struct {
	a1 *unsafe.Pointer // control address
	o1 unsafe.Pointer  // expected value
	a2 *unsafe.Pointer // data address
	o2 unsafe.Pointer  // old value
	n  unsafe.Pointer  // new value
}

// RDCSS performs a Double-Compare Single-Swap atomic
// operation. It attempts to change data address pointer
// `a2` to a `rdcssDescriptor` by comparing it against
// old value `o2`. When successfull, the pointer is changed
// to new value `n` or re-instiated to `o2` in case of
// unsuccessfull operation; A descriptor is active when
// referenced from `a2`. Pointer tagging is used to distinct
// `rdcssDescriptor` pointers.
func RDCSS(a1 *unsafe.Pointer, o1 unsafe.Pointer, a2 *unsafe.Pointer, o2 unsafe.Pointer, n unsafe.Pointer) bool {
	// Paper: A Practical Multi-Word Compare-and-Swap Operation
	//        by Timothy L. Harris, Keir Fraser and Ian A. Pratt;
	//        University of Cambridge Computer Laboratory, Cambridge,
	//        UK.
	var (
		desc *rdcssDescriptor = &rdcssDescriptor{a1, o1, a2, o2, n}
		dptr unsafe.Pointer
	)
	// add `0x1` tag
	dptr, _ = TaggedPointer(unsafe.Pointer(desc), 1)
	if atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(desc.a2)),
		(unsafe.Pointer)(desc.o2),
		(unsafe.Pointer)(dptr),
	) {
		return RDCSSComplete(desc)
	}
	return false
}

// RDCSSComplete performs the second stage when descriptor
// is succesfully stored in `a2`. It finishes the operation
// by swapping `a2` with target pointer `n`. The operation
// is successfull, when `a2` is not pointing to rdcssDescriptor.
// In case of unsucessfull operation, `a2` is swapped with `o2` and
// returns false. Note, `rdcssDescriptor` pointers have a 0x1
// tag attached to low-order bits.
func RDCSSComplete(d *rdcssDescriptor) bool {
	var (
		desc   *rdcssDescriptor
		tgdptr unsafe.Pointer
		dptr   unsafe.Pointer = (atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&d))))
	)
	// used to compare against tagged
	// `rdcssDescriptor` pointer.
	tgdptr, _ = TaggedPointer(dptr, 1)
	desc = (*rdcssDescriptor)(Untag(dptr))
	if (*desc.a1 == desc.o1) && atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(desc.a2)),
		(unsafe.Pointer)(unsafe.Pointer(tgdptr)),
		(unsafe.Pointer)(desc.n),
	) {
		return true
	}
	if !atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(desc.a2)),
		(unsafe.Pointer)(tgdptr),
		(unsafe.Pointer)(desc.o2),
	) {
		// TODO
		log.Println("[-] restoration failed.")
	}
	return false
}

// IsRDCSSDescriptor checks whether the given pointer
// `addr` is pointong to `rdcssDescriptor`or not. According
// to original paper ( Section 6.2 ), `rdcssDescriptor`
// pointers can be made distinct by non-zero low-order
// bits. A pointer is pointing to `rdcssDescriptor` iff
// `0x1` is present.
func IsRDCSSDescriptor(addr unsafe.Pointer) bool {
	return HasTag(addr)
}

// - MARK: Utility section.

// roundP2 rounds the given number `v` to nearest
// power of 2.
func roundP2(v uint64) uint64 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	v++

	return v
}
