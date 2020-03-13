// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

package leveldb_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethersphere/swarm/chunk"
	chunktesting "github.com/ethersphere/swarm/chunk/testing"
	"github.com/ethersphere/swarm/storage/fcds"
	"github.com/ethersphere/swarm/storage/fcds/leveldb"
	"github.com/ethersphere/swarm/storage/fcds/test"
)

// TestFCDS runs a standard series of tests on main Store implementation
// with LevelDB meta store.
func TestFCDS(t *testing.T) {
	test.RunAll(t, func(t *testing.T) (fcds.Storer, func()) {
		path, err := ioutil.TempDir("", "swarm-fcds-")
		if err != nil {
			t.Fatal(err)
		}

		metaStore, err := leveldb.NewMetaStore(filepath.Join(path, "meta"))
		if err != nil {
			t.Fatal(err)
		}

		return test.NewFCDSStore(t, path, metaStore)
	})
}

// TestFreeSlotCounter tests that the free slot counter gets persisted
// and properly loaded on existing store restart
func TestFreeSlotCounter(t *testing.T) {
	path, err := ioutil.TempDir("", "swarm-fcds-")
	if err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(path, "meta")

	metaStore, err := leveldb.NewMetaStore(metaPath)
	if err != nil {
		t.Fatal(err)
	}

	store, err := fcds.New(path, chunk.DefaultSize, metaStore, fcds.WithCache(false))
	if err != nil {
		os.RemoveAll(path)
		t.Fatal(err)
	}

	defer func() {
		store.Close()
		os.RemoveAll(path)
	}()

	// put some chunks, delete some chunks, find the free slots
	// then close the store, init a new one on the same dir
	// then check free slots again and compare
	numChunks := 100
	deleteChunks := 10
	chunks := make([]chunk.Chunk, numChunks)

	for i := 0; i < numChunks; i++ {
		chunks[i] = chunktesting.GenerateTestRandomChunk()
		_, err := store.Put(chunks[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < deleteChunks; i++ {
		err := store.Delete(chunks[i].Address())
		if err != nil {
			t.Fatal(err)
		}
	}

	// verify free slots
	cnt := 0

	metaStore.IterateFree(func(uint8, int64) {
		cnt++
	})

	if cnt != 10 {
		t.Fatalf("expected %d free slots but got %d", 10, cnt)
	}

	store.Close()
	metaStore.Close()

	metaStore2, err := leveldb.NewMetaStore(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		metaStore2.Close()
		os.RemoveAll(metaPath)
	}()

	cnt = 0

	metaStore2.IterateFree(func(_ uint8, _ int64) {
		cnt++
	})

	if cnt != 10 {
		t.Fatalf("expected %d free slots but got %d", 10, cnt)
	}
}