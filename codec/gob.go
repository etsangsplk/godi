package codec

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"io"
	"strings"

	"github.com/Byron/godi/api"
)

const (
	GobName      = "gob"
	GobExtension = "gobz"
	Version      = 1
)

// Reads and writes a file structured like so
// - version
// - numEntries
// - gobValue...
// - sha1 (hash of all hashes in prior map)
type Gob struct {
}

func (g *Gob) Extension() string {
	return GobExtension
}

func (g *Gob) Serialize(in <-chan api.FileInfo, writer io.Writer) (err error) {
	gzipWriter, _ := gzip.NewWriterLevel(writer, 9)
	defer gzipWriter.Close()
	encoder := gob.NewEncoder(gzipWriter)

	sha1enc := sha1.New()

	if err = encoder.Encode(Version); err != nil {
		return
	}

	// NOTE: we re-encode to get rid of the map
	for finfo := range in {
		hashInfo(sha1enc, &finfo)
		if err = encoder.Encode(&finfo); err != nil {
			return
		}
	}

	// Write a marker which will tell that the block of fileInfos is done.
	// That way, when reading, we will get an error once, and are ready to read
	// the final hash
	if err = encoder.Encode(true); err != nil {
		return
	}

	if err = encoder.Encode(sha1enc.Sum(nil)); err != nil {
		return
	}

	return
}

func (g *Gob) Deserialize(reader io.Reader, out chan<- api.FileInfo, predicate func(*api.FileInfo) bool) error {
	// formats an error to match our desired type
	fe := func(err error) error {
		return &DecodeError{Msg: err.Error()}
	}

	gzipReader, _ := gzip.NewReader(reader)
	sha1enc := sha1.New()
	d := gob.NewDecoder(gzipReader)

	// Lets make the fields clear, and not reuse variables even if we could
	fileVersion := 0
	if err := d.Decode(&fileVersion); err != nil {
		return fe(err)
	}

	// Of course we would implement reading other formats too
	if fileVersion != Version {
		return &DecodeError{Msg: fmt.Sprintf("Cannot handle index file: invalid header version: %d", fileVersion)}
	}

	var readError error
	for readError == nil {
		// Yes - we need a fresh one every loop iteration ! Gob doesn't set fields which have the nil value
		v := api.FileInfo{}

		// If there is a type-mismatch, we are done reading values and proceed with final signature check
		if readError = d.Decode(&v); readError != nil {
			// Unfortunately, we can't really tell programmatically what happened - need to rely on string scanning :(
			if strings.Contains(readError.Error(), "type mismatch in decoder") {
				break
			} else {
				return fe(readError)
			}
		}

		// Have to hash it before we hand it to the predicate, as it might alter the data
		hashInfo(sha1enc, &v)

		if !predicate(&v) {
			return nil
		}
		out <- v
	}

	var signature []byte
	if err := d.Decode(&signature); err != nil {
		return fe(err)
	}

	// Finally, compare signature of seal with the one we made ...
	if bytes.Compare(signature, sha1enc.Sum(nil)) != 0 {
		return &SignatureMismatchError{}
	}

	return nil
}
