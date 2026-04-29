// storage_zstd.go â€” ZSTD compression layer for KVStore ciphertext storage.
//
// Architecture:
//   TFHE-rs ciphertext (~21KB)
//   â†’ tfhe_compress (TFHE post-computation compression)
//   â†’ zstdCompress  (ZSTD BestCompression)
//   = stored bytes  (~5KB)
//
//   On read: reverse â€” zstdDecompress â†’ tfhe_decompress â†’ homomorphic ops.
//
// This keeps the fhe.Context and homomorphic arithmetic completely clean
// (they always work on TFHE-rs native bytes). ZSTD is a pure storage
// optimization applied only at the KVStore boundary.
package keeper

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// zstd encoder/decoder are safe for concurrent use once initialised.
var (
	zstdEncoderOnce sync.Once
	zstdDecoderOnce sync.Once
	zstdEncoder     *zstd.Encoder
	zstdDecoder     *zstd.Decoder
	zstdInitErr     error
)

func getZstdEncoder() (*zstd.Encoder, error) {
	zstdEncoderOnce.Do(func() {
		enc, err := zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.SpeedBestCompression),
			zstd.WithEncoderConcurrency(1),
		)
		if err != nil {
			zstdInitErr = fmt.Errorf("zstd: failed to create encoder: %w", err)
			return
		}
		zstdEncoder = enc
	})
	if zstdInitErr != nil {
		return nil, zstdInitErr
	}
	return zstdEncoder, nil
}

func getZstdDecoder() (*zstd.Decoder, error) {
	zstdDecoderOnce.Do(func() {
		dec, err := zstd.NewReader(nil,
			zstd.WithDecoderConcurrency(0),
		)
		if err != nil {
			zstdInitErr = fmt.Errorf("zstd: failed to create decoder: %w", err)
			return
		}
		zstdDecoder = dec
	})
	if zstdInitErr != nil {
		return nil, zstdInitErr
	}
	return zstdDecoder, nil
}

// zstdCompressForStore compresses src bytes using ZSTD BestCompression.
// Applied on top of TFHE-rs built-in compression at the KVStore write boundary.
func zstdCompressForStore(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("zstd: cannot compress empty input")
	}
	enc, err := getZstdEncoder()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	// EncodeAll is safe to call concurrently.
	compressed := enc.EncodeAll(src, buf.Bytes())
	return compressed, nil
}

// zstdDecompressFromStore decompresses src bytes using ZSTD.
// Applied before TFHE-rs decompress at the KVStore read boundary.
func zstdDecompressFromStore(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("zstd: cannot decompress empty input")
	}
	dec, err := getZstdDecoder()
	if err != nil {
		return nil, err
	}
	decompressed, err := dec.DecodeAll(src, nil)
	if err != nil {
		return nil, fmt.Errorf("zstd: decompress failed: %w", err)
	}
	return decompressed, nil
}