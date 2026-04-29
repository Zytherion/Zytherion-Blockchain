//go:build !notfhe
// +build !notfhe

package fhe

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ltfhe_cgo -ldl -lm -lpthread
#cgo CFLAGS: -I${SRCDIR}/lib
#include "tfhe_cgo.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func tfheGenerateKeys() (clientKeyBytes []byte, serverKeyBytes []byte, err error) {
	var ckPtr *C.uint8_t
	var ckLen C.size_t
	var skPtr *C.uint8_t
	var skLen C.size_t
	rc := C.tfhe_generate_keys(&ckPtr, &ckLen, &skPtr, &skLen)
	if rc != 0 {
		return nil, nil, fmt.Errorf("tfhe: key generation failed")
	}
	ck := C.GoBytes(unsafe.Pointer(ckPtr), C.int(ckLen))
	C.tfhe_free_bytes(ckPtr, ckLen)
	sk := C.GoBytes(unsafe.Pointer(skPtr), C.int(skLen))
	C.tfhe_free_bytes(skPtr, skLen)
	return ck, sk, nil
}

func tfheEncrypt(clientKey []byte, value uint64) ([]byte, error) {
	var outPtr *C.uint8_t
	var outLen C.size_t
	ckPtr := (*C.uint8_t)(unsafe.Pointer(&clientKey[0]))
	rc := C.tfhe_encrypt_u64(ckPtr, C.size_t(len(clientKey)), C.uint64_t(value), &outPtr, &outLen)
	if rc != 0 {
		return nil, fmt.Errorf("tfhe: encrypt failed")
	}
	result := C.GoBytes(unsafe.Pointer(outPtr), C.int(outLen))
	C.tfhe_free_bytes(outPtr, outLen)
	return result, nil
}

func tfheDecrypt(clientKey []byte, ct []byte) (uint64, error) {
	var outValue C.uint64_t
	ckPtr := (*C.uint8_t)(unsafe.Pointer(&clientKey[0]))
	ctPtr := (*C.uint8_t)(unsafe.Pointer(&ct[0]))
	rc := C.tfhe_decrypt_u64(ckPtr, C.size_t(len(clientKey)), ctPtr, C.size_t(len(ct)), &outValue)
	if rc != 0 {
		return 0, fmt.Errorf("tfhe: decrypt failed")
	}
	return uint64(outValue), nil
}

func tfheAdd(serverKey []byte, ctA []byte, ctB []byte) ([]byte, error) {
	var outPtr *C.uint8_t
	var outLen C.size_t
	skPtr := (*C.uint8_t)(unsafe.Pointer(&serverKey[0]))
	aPtr := (*C.uint8_t)(unsafe.Pointer(&ctA[0]))
	bPtr := (*C.uint8_t)(unsafe.Pointer(&ctB[0]))
	rc := C.tfhe_add(skPtr, C.size_t(len(serverKey)),
		aPtr, C.size_t(len(ctA)),
		bPtr, C.size_t(len(ctB)),
		&outPtr, &outLen)
	if rc != 0 {
		return nil, fmt.Errorf("tfhe: homomorphic add failed")
	}
	result := C.GoBytes(unsafe.Pointer(outPtr), C.int(outLen))
	C.tfhe_free_bytes(outPtr, outLen)
	return result, nil
}

func tfheSub(serverKey []byte, ctA []byte, ctB []byte) ([]byte, error) {
	var outPtr *C.uint8_t
	var outLen C.size_t
	skPtr := (*C.uint8_t)(unsafe.Pointer(&serverKey[0]))
	aPtr := (*C.uint8_t)(unsafe.Pointer(&ctA[0]))
	bPtr := (*C.uint8_t)(unsafe.Pointer(&ctB[0]))
	rc := C.tfhe_sub(skPtr, C.size_t(len(serverKey)),
		aPtr, C.size_t(len(ctA)),
		bPtr, C.size_t(len(ctB)),
		&outPtr, &outLen)
	if rc != 0 {
		return nil, fmt.Errorf("tfhe: homomorphic sub failed")
	}
	result := C.GoBytes(unsafe.Pointer(outPtr), C.int(outLen))
	C.tfhe_free_bytes(outPtr, outLen)
	return result, nil
}

func tfheCompress(serverKey []byte, ct []byte) ([]byte, error) {
	var outPtr *C.uint8_t
	var outLen C.size_t
	skPtr := (*C.uint8_t)(unsafe.Pointer(&serverKey[0]))
	ctPtr := (*C.uint8_t)(unsafe.Pointer(&ct[0]))
	rc := C.tfhe_compress(skPtr, C.size_t(len(serverKey)), ctPtr, C.size_t(len(ct)), &outPtr, &outLen)
	if rc != 0 {
		return nil, fmt.Errorf("tfhe: compress failed")
	}
	result := C.GoBytes(unsafe.Pointer(outPtr), C.int(outLen))
	C.tfhe_free_bytes(outPtr, outLen)
	return result, nil
}

func tfheDecompress(serverKey []byte, compressed []byte) ([]byte, error) {
	var outPtr *C.uint8_t
	var outLen C.size_t
	skPtr := (*C.uint8_t)(unsafe.Pointer(&serverKey[0]))
	cPtr := (*C.uint8_t)(unsafe.Pointer(&compressed[0]))
	rc := C.tfhe_decompress(skPtr, C.size_t(len(serverKey)), cPtr, C.size_t(len(compressed)), &outPtr, &outLen)
	if rc != 0 {
		return nil, fmt.Errorf("tfhe: decompress failed")
	}
	result := C.GoBytes(unsafe.Pointer(outPtr), C.int(outLen))
	C.tfhe_free_bytes(outPtr, outLen)
	return result, nil
}