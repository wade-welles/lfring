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

// lfring provides Lock-Free Multi-Reader, Multi-Writer Ring Buffer implementation.
package lfring

import (
	"errors"
	"unsafe"
)

// Defaults
const (
	// archADDRSIZE is either 32 or 64
	archADDRSIZE = 32 << uintptr(^uintptr(0)>>63)
	// archWORDSIZE is either 4 or 8
	archWORDSIZE = archADDRSIZE >> 3
	// archMAXTAG is either 3 or 7
	archMAXTAG = archWORDSIZE - 1
	// archPTRMASK is either 2 or 3
	archPTRMASK = ^uintptr((archADDRSIZE >> 5) + 1)
	// ui64MASK is maximum int value
	ui64NMASK = ^uint64(0)
	// cRDSCHDTHRESHOLD is reader's spin threshold before
	// yielding control with `runtime.Gosched()`.
	cRDSCHDTHRESHOLD = 1000
	// cRDSCHDTHRESHOLD is writer's spin threshold
	// before yielding control with `runtime.Gosched()`.
	cWRSCHDTHRESHOLD = 1000
)

// Errors
var (
	EPTRINVAL  error = errors.New("pointer: invalid.")
	EPTRINVALT error = errors.New("pointer: invalid tag.")
)

// Helpers
var (
	_PTR_       unsafe.Pointer
	_INTERFACE_ interface{}
	// archPTRSIZE is pointer size
	archPTRSIZE uintptr = unsafe.Sizeof(_PTR_)
	// sizeINTERFACE is interface size
	sizeINTERFACE uintptr = unsafe.Sizeof(_INTERFACE_)
)
