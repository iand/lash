# lash

Go package providing a persistent, concurrent, memory-resident key/value hashtable.

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/iand/lash)
[![Check Status](https://github.com/iand/lash/actions/workflows/check.yml/badge.svg)](https://github.com/iand/lash/actions/workflows/check.yml)
[![Test Status](https://github.com/iand/lash/actions/workflows/test.yml/badge.svg)](https://github.com/iand/lash/actions/workflows/test.yml)


## Overview

Lash provides Table, a persistent, concurrent, memory-resident key/value hashtable. It is designed to persist its state on disk and recover it in the event of a crash or restart. It uses a log-based approach to data storage. Each key and value are appended to the underlying data file before being inserted into the memory hashtable. Data to be deleted from the table is marked with a tombstone in the data file. Tombstones are evicted when restoring the table from the data file during initialisation. This simple log-based approach performs well but will lead to very large data files for long-lived tables with high volumes of writes. Currently the only method of compacting the data file is to close the table and instantiate a new one pointing at the same file. 

Note: this package is considered to be in an alpha state. The happy path works well but there are
dozens of potential corner cases around its I/O that need to be figured out.

## Usage

```Go
package main

import (
    "github.com/iand/lash"
    "log"
)

func main() {
    table, err := lash.New("data.db", 50)
    if err != nil {
        log.Fatal(err.Error())
    }
    defer table.Close()

    err = table.Put("key1", []byte("value"))
    if err != nil {
        log.Fatal(err.Error())
    }

    v, found := table.Get("key1")
    if !found {
        log.Fatal("did not find key")
    }
    log.Printf("%s", v)
}   
```

## Installation

Simply run

	go get github.com/iand/lash

Documentation is at [http://godoc.org/github.com/iand/lash](http://godoc.org/github.com/iand/lash)

## License

This is free and unencumbered software released into the public domain. For more
information, see <http://unlicense.org/> or the accompanying [`LICENSE`](LICENSE) file.
