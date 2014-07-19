package seal

import (
	"github.com/Byron/godi/api"
)

const (
	IndexBaseName = "godi"
	Name          = "seal"
)

// A type representing all arguments required to drive a Seal operation
type SealCommand struct {

	// One or more trees to seal
	Trees []string

	// Amount of readers to use
	nReaders int

	// parallel reader
	pCtrl api.ReadChannelController
}

// Implements information about a seal operation
type SealResult struct {
	finfo *api.FileInfo
	msg   string
	err   error
	prio  api.Priority
}

// REVIEW:
func NewCommand(trees []string, nReaders int) SealCommand {
	c := SealCommand{}
	c.Trees = trees
	c.nReaders = nReaders
	return c
}
