/*
  This is free and unencumbered software released into the public domain. For more
  information, see <http://unlicense.org/> or the accompanying UNLICENSE file.
*/

// Package lash provides a persistent, concurrent, memory-resident key/value hashtable.
package lash

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
)

// New creates a new Table backed by the file fname and with an initial capacity
// of n. If filename is an empty string then the table will not persist data
// and will operate purely in memory as though it were a less performant, but
// concurrent version of the Go map type. If the file fname already exists then
// it will be read to initialise the data for the table, compacting the file
// in the process by rewriting it to remove tombstones.
func New(fname string, n int) (*Table, error) {
	t := &Table{
		data:     make(map[string]item, n),
		filename: fname,
	}

	return t, t.read()
}

type item struct {
	val []byte
	pos int64
}

// Table is a persistent, concurrent, memory-resident key/value hashtable.
// It is designed to persist its state on disk and recover it in the event
// of a crash or restart. It uses a log-based approach to data storage. Each
// key and value are appended to the underlying data file before being inserted
// into the memory hashtable. Data to be deleted from the table is marked
// with a tombstone in the data file. Tombstones are evicted when restoring
// the table from the data file during initialisation. This simple log-based
// approach performs well but will lead to very large data files for long-lived
// tables with high volumes of writes. Currently the only method of compacting
// the data file is to close the table and instantiate a new one pointing at the
// same file.
type Table struct {
	mtx      sync.RWMutex
	data     map[string]item
	filename string
	dbfile   *os.File
}

const sep = byte(31)
const tomb = byte(127)

// write serialises the key and item to the table's datafile
// It returns the file offset at which the data was written
// and/or any error that occurred while writing.
func (t *Table) write(k string, p item) (int64, error) {
	if t.dbfile == nil {
		if t.filename == "" {
			return 0, nil
		}
		return 0, errors.New("database not open")
	}

	// TODO: sanitize k for tabs
	buf := &bytes.Buffer{}
	buf.Write([]byte(k))
	buf.WriteByte(sep)

	b := []byte(p.val)
	lbuf := make([]byte, binary.MaxVarintLen64)
	lbufn := binary.PutVarint(lbuf, int64(len(b)))
	buf.Write(lbuf[:lbufn])
	buf.Write(b)

	pos, err := t.dbfile.Seek(0, os.SEEK_END)
	if err != nil {
		return 0, err
	}

	// TODO: check number of bytes written
	n, err := buf.WriteTo(t.dbfile)
	if err != nil {
		if n == 0 {
			return 0, err
		}
		// TODO: decide what to do on a partial write error
		return 0, err
	}

	err = t.dbfile.Sync()
	if err != nil {
		return pos, err
	}

	return pos, nil
}

// mark inserts a tombstone marker in the data file for a deleted item
func (t *Table) mark(pos int64) error {
	if t.dbfile == nil {
		if t.filename == "" {
			return nil
		}
		return errors.New("database not open")
	}

	// TODO: check number of bytes written
	_, err := t.dbfile.WriteAt([]byte{tomb}, pos)
	if err != nil {
		return err
	}
	return nil
}

func (t *Table) read() error {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.filename == "" {
		return nil
	}

	err := os.Rename(t.filename, t.filename+".swp")
	if err != nil {
		return err
	}

	t.dbfile, err = os.OpenFile(t.filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, os.FileMode(0666))
	if err != nil {
		return err
	}

	swapFile, err := os.Open(t.filename + ".swp")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer os.Remove(swapFile.Name())
	defer swapFile.Close()

	r := bufio.NewReader(swapFile)

	for {
		key, err := r.ReadString(sep)
		if err != nil {
			break
		}

		lb, err := binary.ReadVarint(r)
		if err != nil {
			return err
		}

		buf := make([]byte, lb)
		n, err := io.ReadFull(r, buf)
		if err != nil {
			return err
		}

		if key[0] != tomb {
			t.putnew(key[:len(key)-1], item{val: buf[:n]})
		}
	}

	// TODO: tighten this up, ensure we have read a full key
	if err != io.EOF {
		return err
	}

	return nil
}

// Close closes the underlying data file (if any) for the table. The
// table will continue to respond to read-only methods such as Get and
// Len but will return an error for any mutating methods such as Put.
func (t *Table) Close() error {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.dbfile == nil {
		if t.filename == "" {
			return nil
		}
		return errors.New("database not open")
	}
	return t.dbfile.Close()
}

// Put stores the value v under key k in the table
// and writes it to persistent storage. Any error encountered
// while persisting the data will be returned. If the table fails
// to persist the data then the table will be restored to the state
// it had just prior to the call to Put.
func (t *Table) Put(k string, v []byte) error {
	add := item{
		val: v,
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()

	old, exists := t.data[k]
	if !exists {
		return t.putnew(k, add)
	}

	var err error
	add.pos, err = t.write(k, add)
	if err != nil {
		return err
	}

	t.data[k] = add
	err = t.mark(old.pos)
	if err != nil {
		t.data[k] = old
		return err
	}
	return nil
}

// putnew adds a new item to the table without checking
// whether it is overwriting any existing data.
// It is the responsibility of the caller to acquire locks.
func (t *Table) putnew(k string, add item) error {
	var err error
	add.pos, err = t.write(k, add)
	if err != nil {
		return err
	}
	t.data[k] = add
	return nil
}

// Get retrieves the value stored under key k and returns it
// along with a boolean that indicates whether the value was
// found in the table or not.
func (t *Table) Get(k string) ([]byte, bool) {
	t.mtx.RLock()
	cur, found := t.data[k]
	t.mtx.RUnlock()
	return cur.val, found
}

// Len returns the number of items in the table.
func (t *Table) Len() int {
	t.mtx.RLock()
	l := len(t.data)
	t.mtx.RUnlock()
	return l
}
