package buffer

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"
)

func intEntry(i int) *entry.Entry {
	e := entry.New()
	e.Timestamp = time.Date(2006, 01, 02, 03, 04, 05, 06, time.UTC)
	e.Record = float64(i)
	return e
}

func writeN(t testing.TB, buffer Buffer, n, start int) {
	ctx := context.Background()
	for i := start; i < n+start; i++ {
		err := buffer.Add(ctx, intEntry(i))
		require.NoError(t, err)
	}
}

func readN(t testing.TB, buffer Buffer, n, start int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, err := buffer.Read(entries)
	require.NoError(t, err)
	require.Equal(t, n, readCount)
	for i := 0; i < n; i++ {
		require.Equal(t, intEntry(start+i), entries[i])
	}
	return f
}

func readWaitN(t testing.TB, buffer Buffer, n, start int) func() {
	entries := make([]*entry.Entry, n)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	f, readCount, err := buffer.ReadWait(ctx, entries)
	require.NoError(t, err)
	require.Equal(t, n, readCount)
	for i := 0; i < n; i++ {
		require.Equal(t, intEntry(start+i), entries[i])
	}
	return f
}

func uncheckedReadN(t testing.TB, buffer Buffer, n int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, _ := buffer.Read(entries)
	require.Equal(t, n, readCount)
	return f
}

func flushN(t testing.TB, buffer Buffer, n, start int) {
	f := readN(t, buffer, n, start)
	f()
}

func uncheckedFlushN(t testing.TB, buffer Buffer, n int) {
	f := uncheckedReadN(t, buffer, n)
	f()
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}