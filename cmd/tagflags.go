package main

import (
	"strings"

	"github.com/srerickson/bago"
)

type TagFlags bago.TagFile

// String returns string representation of the tag file following the
// the BagIt specification. Lines are wrapped
func (tf *TagFlags) String() string {
	var builder strings.Builder
	(*bago.TagFile)(tf).Write(&builder)
	return builder.String()
}

// Set is required by the Flag interface so we can collect tag values from the
// command line. It is also used in parse()
func (tf *TagFlags) Set(val string) error {
	vals, err := bago.ParseTagFileLine(val)
	if err != nil {
		return err
	}
	(*bago.TagFile)(tf).Append(vals[0], vals[1])
	return nil
}
