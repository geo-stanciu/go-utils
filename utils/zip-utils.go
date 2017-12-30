package utils

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"sync"
)

// ErrinvalidEntry - invalid entry index
var ErrinvalidEntry = errors.New("invalid entry index")

// ErrEOF - if the end of the zip archive was already reached
var ErrEOF = errors.New("no more entries")

// ErrEntryNotFound - entry not found
var ErrEntryNotFound = errors.New("entry name not found")

// ZipWriter - zip creator helper
type ZipWriter struct {
	sync.RWMutex
	dest io.Writer
	w    *zip.Writer
}

// NewZipWriter - instantiates a ZipWriter
func NewZipWriter(dest io.Writer) *ZipWriter {
	z := ZipWriter{}

	z.dest = dest
	z.w = zip.NewWriter(dest)

	return &z
}

// AddEntry - add file
func (z *ZipWriter) AddEntry(name string, content []byte) error {
	z.Lock()
	defer z.Unlock()

	f, err := z.w.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write(content)
	if err != nil {
		return err
	}

	return nil
}

// Close - closes the archive and makes it ready to use
// must call Close prior trying to using the newly created archive
func (z *ZipWriter) Close() error {
	z.Lock()
	defer z.Unlock()

	err := z.w.Close()
	if err != nil {
		return err
	}

	return nil
}

// ZipReader - ZipReader helper
type ZipReader struct {
	sync.RWMutex
	r            *zip.Reader
	nrEntries    int
	entries      []string
	currentEntry int
}

// NewZipReader - instantiates a new ZipReader
func NewZipReader(zipcontent []byte) (*ZipReader, error) {
	z := ZipReader{}

	n := int64(len(zipcontent))
	r, err := zip.NewReader(bytes.NewReader(zipcontent), n)
	if err != nil {
		return nil, err
	}

	z.r = r
	z.currentEntry = -1

	z.nrEntries = len(z.r.File)
	z.entries = make([]string, z.nrEntries)
	for i, f := range z.r.File {
		z.entries[i] = f.Name
	}

	return &z, nil
}

// GetEntries - get file names
func (z *ZipReader) GetEntries() []string {
	z.RLock()
	defer z.RUnlock()

	return z.entries
}

// GetEntry - get file content
func (z *ZipReader) GetEntry(name string, dest io.Writer) error {
	z.Lock()
	defer z.Unlock()

	for i, f := range z.r.File {
		if name == f.Name {
			err := z.readAtIndex(i, dest)
			if err != nil {
				return err
			}
		}
	}

	return ErrEntryNotFound
}

// ReadCurrentEntry - get current entry content
func (z *ZipReader) ReadCurrentEntry(dest io.Writer) error {
	z.Lock()
	defer z.Unlock()
	return z.readAtIndex(z.currentEntry, dest)
}

func (z *ZipReader) readAtIndex(i int, dest io.Writer) error {
	if i < 0 {
		return ErrinvalidEntry
	}

	if i > z.nrEntries-1 {
		return ErrEOF
	}

	f := z.r.File[i]

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(dest, rc)
	if err != nil {
		return err
	}

	return nil
}

// HasNextEntry - tests if there are more files to get
func (z *ZipReader) HasNextEntry() bool {
	z.Lock()
	defer z.Unlock()

	if z.currentEntry >= z.nrEntries-1 {
		return false
	}

	return true
}

// GetNextEntry - gets the next file name
func (z *ZipReader) GetNextEntry() (string, error) {
	z.Lock()
	defer z.Unlock()

	if z.currentEntry >= z.nrEntries-1 {
		return "", ErrEOF
	}

	z.currentEntry++
	return z.r.File[z.currentEntry].Name, nil
}

// ResetEntryIndex - resets current index to the first file
func (z *ZipReader) ResetEntryIndex() {
	z.Lock()
	defer z.Unlock()

	z.currentEntry = -1
}

// Close - free
func (z *ZipReader) Close() error {
	z.Lock()
	defer z.Unlock()

	return nil
}
