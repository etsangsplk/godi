package seal

import (
	"github.com/Byron/godi/api"
	"github.com/Byron/godi/io"
)

const (
	Name = "seal"

	modeSeal = Name
	modeCopy = "sealed-copy"
)

type indexWriterResult struct {
	path string // path to the seal file
	err  error  // possible error during the seal operation
}

// Some information we store per root of files we seal
type aggregationTreeInfo struct {
	// Paths to files we have written so far - only used in sealed-copy mode
	// TODO(st): don't track these files in memory, but re-read them from the written seal file !
	// That way, we don't rely on any limited resource except for disk space
	writtenFiles []string

	// A channel to send file-infos to the attached seal serializer. Close it to finish the seal operation
	sealFInfos chan<- api.FileInfo

	// Contains the error code of the seal operation for the tree we are associated with, and the produced seal file
	// Will only yield a result one, and be closed afterwards
	sealResult <-chan indexWriterResult

	// A possible result we might have gotten due to an early seal error
	lsr indexWriterResult // lastSealResult

	// if true, the entire tree is considered faulty, and further results won't be recorded or accepted
	hasError bool
}

// Helper to sort by longest path, descending
type byLongestPathDescending []string

func (a byLongestPathDescending) Len() int           { return len(a) }
func (a byLongestPathDescending) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byLongestPathDescending) Less(i, j int) bool { return len(a[i]) > len(a[j]) }

// A type representing all arguments required to drive a Seal operation
type SealCommand struct {
	api.BasicRunner

	// The type of seal operation we are supposed to perform
	mode string

	// If set, we are supposed to run in verify mode
	verify bool

	// The name of the seal format to use
	format string

	// A map of writers - there may just be one writer per device.
	// Map may be unset if we are not in write mode
	rootedWriters []io.RootedWriteController
}

// A result which is also able to hold information about the source of a file
type SealResult struct {
	api.BasicResult
	// source of a copy operation, may be unset
	source string
}

// Returns true if this result was sent from a generator. The latter sends the root as Path, but doesn't set a RelaPath
func (s *SealResult) FromGenerator() bool {
	return len(s.Finfo.RelaPath) == 0
}

// NewCommand returns an initialized seal command
func NewCommand(trees []string, nReaders, nWriters int) (*SealCommand, error) {
	c := SealCommand{}
	if nWriters == 0 {
		c.mode = modeSeal
	} else {
		c.mode = modeCopy
	}
	err := c.Init(nReaders, nWriters, trees, api.Info, []api.FileFilter{api.FilterSeals})
	return &c, err
}

func (s *SealCommand) Gather(rctrl *io.ReadChannelController, files <-chan api.FileInfo, results chan<- api.Result) {
	makeResult := func(f, source *api.FileInfo, err error) api.Result {
		s := ""
		if source != nil {
			s = source.Path
		}
		res := SealResult{
			BasicResult: api.BasicResult{
				Finfo: *f,
				Prio:  api.Info,
				Err:   err,
			},
			source: s,
		}
		return &res
	}

	api.Gather(files, results, s.Statistics(), makeResult, rctrl, s.rootedWriters)
}
