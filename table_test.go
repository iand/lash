/*
  This is free and unencumbered software released into the public domain. For more
  information, see <http://unlicense.org/> or the accompanying UNLICENSE file.
*/

package lash

import (
	"io/ioutil"
	"os"
	"testing"
)

func makeTable(n int) (*Table, *os.File, error) {
	tf, err := ioutil.TempFile("", "lash")
	if err != nil {
		return nil, nil, err
	}
	tf.Close()

	table, err := New(tf.Name(), n)
	if err != nil {
		return nil, nil, err
	}

	return table, tf, nil
}

func TestPut(t *testing.T) {
	table, tf, err := makeTable(50)
	if err != nil {
		t.Fatal(err.Error())

	}
	defer os.Remove(tf.Name())
	defer table.Close()

	err = table.Put("a", []byte("val"))
	if err != nil {
		t.Fatal(err.Error())
	}

	v, found := table.Get("a")
	if !found {
		t.Fatalf("got not found, wanted found")
	}
	if string(v) != "val" {
		t.Errorf("got %q, wanted %q", v, "val")
	}
}

func TestPutMem(t *testing.T) {
	table, err := New("", 50)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer table.Close()

	err = table.Put("a", []byte("val"))
	if err != nil {
		t.Fatal(err.Error())
	}

	v, found := table.Get("a")
	if !found {
		t.Fatalf("got not found, wanted found")
	}
	if string(v) != "val" {
		t.Errorf("got %q, wanted %q", v, "val")
	}
}

func TestPutEvict(t *testing.T) {
	table, tf, err := makeTable(50)
	if err != nil {
		t.Fatal(err.Error())

	}
	defer os.Remove(tf.Name())
	defer table.Close()

	err = table.Put("a", []byte("val"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = table.Put("a", []byte("val2"))
	if err != nil {
		t.Fatal(err.Error())
	}

	v, found := table.Get("a")
	if !found {
		t.Fatalf("got not found, wanted found")
	}
	if string(v) != "val2" {
		t.Errorf("got %q, wanted %q", v, "val")
	}
}

func TestRead(t *testing.T) {
	// Create a table and put an entry in it
	table, tf, err := makeTable(50)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = table.Put("a", []byte("val"))
	if err != nil {
		t.Fatal(err.Error())
	}
	table.Close()

	// Open a table from same filename
	table2, err := New(tf.Name(), 50)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.Remove(tf.Name())
	defer table.Close()

	v, found := table2.Get("a")
	if !found {
		t.Fatalf("got not found, wanted found")
	}
	if string(v) != "val" {
		t.Errorf("got %q, wanted %q", v, "val")
	}
}
