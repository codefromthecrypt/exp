package main

import (
	"reflect"
	"unsafe"
)

func main() {}

//export memory_allocate
func memoryAllocate(size uint32) *byte {
	b := make([]byte, size)
	return &b[0]
}

//export get_message
func getMessage(ptr **byte, size *uintptr)

//export allocate_message
func allocateMessage() uint64 {
	var ptr *byte
	var size uintptr

	getMessage(&ptr, &size)

	buf := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(ptr)),
		Len:  size,
		Cap:  size,
	}))

	allocationCounter += uint64(len(buf))
	return allocationCounter
}

var allocationCounter uint64
