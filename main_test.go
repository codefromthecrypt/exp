package main

import (
	"context"
	_ "embed"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:generate tinygo build -target=wasi -scheduler=none -o ./testdata/main.wasm ./testdata/main.go
//go:embed testdata/main.wasm
var bin []byte

func TestGCWorks(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	require.NoError(t, err)

	const allocationBlock = 1000

	var memoryAllocate api.Function
	var memory api.Memory
	_, err = r.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(dataPtrPtr uint32, sizePtr uint32) {
			res, err := memoryAllocate.Call(ctx, allocationBlock)
			require.NoError(t, err)

			dataPtr := uint32(res[0])

			// Write the random message
			data, ok := memory.Read(ctx, dataPtr, allocationBlock)
			require.True(t, ok)
			_, err = rand.Read(data)
			require.NoError(t, err)

			ok = memory.WriteUint32Le(ctx, dataPtrPtr, dataPtr)
			require.True(t, ok)
			ok = memory.WriteUint32Le(ctx, sizePtr, allocationBlock)
			require.True(t, ok)
		}).
		Export("get_message").
		Instantiate(ctx, r)
	require.NoError(t, err)

	compiled, err := r.CompileModule(ctx, bin)
	require.NoError(t, err)

	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithStdout(os.Stdout))
	require.NoError(t, err)

	memory = mod.Memory()
	memoryAllocate = mod.ExportedFunction("memory_allocate")
	allocateMessage := mod.ExportedFunction("allocate_message")

	sizeBefore := memory.Size(ctx)

	var allocatedBytes uint64

	// Try invoking allocateMessage enough to have the total allocated bytes exceed the memory size.
	for i := uint64(0); i < (uint64(sizeBefore)/allocationBlock)*100; i++ {
		res, err := allocateMessage.Call(ctx)
		require.NoError(t, err)
		allocatedBytes = res[0]
	}

	// Ensures that the memory doesn't grow.
	sizeAfter := memory.Size(ctx)
	require.Equal(t, sizeBefore, sizeAfter)

	// Ensures that allocate bytes are actually larger than the memory size.
	require.True(t, allocatedBytes > uint64(sizeAfter))
}
