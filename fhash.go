package main

import (
	"io"
	"fmt"
	"os"
	"log"
	"runtime/pprof"
	"crypto/sha512"
	"io/ioutil"
	"path/filepath"
	"encoding/hex"
	"sort"
	"flag"
)

type EntryType byte

const (
	FILE EntryType = 'F'
	DIR EntryType = 'D'
	SUMMARY EntryType = '#'
)

const (
	FHASH_FILENAME = ".fhash"
)

type HashEntry struct {
	Type  EntryType;
	Hash  []byte;
	Size  int64;
	Name  string;
	Files HashEntries;
}

type HashEntries []HashEntry

func hashFile(filename string) (*HashEntry, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	h := sha512.New();

	if _, err := io.Copy(h, file); err != nil {
		return nil, err
	}

	var entry HashEntry

	entry.Type = FILE;
	entry.Hash = h.Sum(nil)
	entry.Size = stat.Size()
	entry.Name = filename

	return &entry, nil
}

func (files HashEntries ) Len() int {
	return len(files)
}

func (files HashEntries ) Less(i, j int) bool {
	return hex.EncodeToString(files[i].Hash) < hex.EncodeToString(files[j].Hash)
}

func (files HashEntries ) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

func hashDirectory(filename string) (*HashEntry, error) {

	dir_entry := &HashEntry{Type: DIR}

	files, err := ioutil.ReadDir(filename)
	if err != nil {
		return nil, err
	}

	for _, file := range files {

		var entry *HashEntry

		if file.Name() == FHASH_FILENAME {
			continue
		}

		if file.IsDir() {
			entry, err = hashDirectory(filepath.Join(filename, file.Name()))
		} else {
			entry, err = hashFile(filepath.Join(filename, file.Name()))
		}
		if err != nil {
			return nil, err
		}

		dir_entry.Files = append(dir_entry.Files, *entry)
		dir_entry.Size += entry.Size
	}

	sort.Sort(dir_entry.Files)

	h := sha512.New()
	for _, e := range dir_entry.Files {
		printHashEntry(h, &e, false, false)
	}
	dir_entry.Hash = h.Sum(nil)

	dir_entry.Name = filepath.Base(filename)

	return dir_entry, nil
}

func printHashEntry(w io.Writer, entry *HashEntry, includeName bool, recursive bool) {
	if ( includeName ) {
		fmt.Fprintf(
			w,
			"%c %s:%d %s\n",
			entry.Type,
			hex.EncodeToString(entry.Hash),
			entry.Size,
			filepath.Base(entry.Name),
		)
	} else {
		fmt.Fprintf(
			w,
			"%c %s:%d\n",
			entry.Type,
			hex.EncodeToString(entry.Hash),
			entry.Size,
		)
	}

	if recursive && entry.Type == DIR {
		for _, e := range entry.Files {
			printHashEntry(w, &e, includeName, false)
		}
	}
}

func main() {

	// profile
	f, err := os.Create("fhash.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	// end of profiling


	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("usage: fhash dir [dir ..]")
		os.Exit(-1)
	}

	for _, arg := range flag.Args() {

		stat, err := os.Stat( arg )
		if err != nil {
			panic( err )
		}

		if ! stat.IsDir() {
			fmt.Fprintln( os.Stderr, "is not a directory: ", arg )
			os.Exit( -1 )
		}

		entry, err := hashDirectory(arg)
		if err != nil {
			panic(err)
		}

		entry.Type = SUMMARY

		fmt.Fprintf(os.Stdout, "%s: %s\n", arg, hex.EncodeToString(entry.Hash))

		f, err := os.Create(filepath.Join(arg, FHASH_FILENAME))
		if err != nil {
			panic(err)
		}
		printHashEntry(f, entry, true, true);

		//printHashEntry(os.Stdout, entry, true, true)
	}
}
