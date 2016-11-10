// Copyright 2016 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package tlf

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/keybase/client/go/libkb"
	"github.com/keybase/go-codec/codec"
)

const (
	// HandleExtensionSep is the string that separates the folder
	// participants from an extension suffix in the TLF name.
	HandleExtensionSep = " "
	// HandleExtensionDateFormat is the date format for the HandleExtension string.
	HandleExtensionDateFormat = "2006-01-02"
	// HandleExtensionDateRegex is the regular expression matching the HandleExtension
	// date member in string form.
	HandleExtensionDateRegex = "2[0-9]{3}-[0-9]{2}-[0-9]{2}"
	// HandleExtensionNumberRegex is the regular expression matching the HandleExtension
	// number member in string form.
	HandleExtensionNumberRegex = "[0-9]+"
	// HandleExtensionUsernameRegex is the regular expression matching the HandleExtension
	// username member in string form.
	HandleExtensionUsernameRegex = "[a-z0-9_]+"
	// HandleExtensionConflictString is the string identifying a conflict extension.
	HandleExtensionConflictString = "conflicted copy"
	// HandleExtensionFinalizedString is the format string identifying a finalized extension.
	HandleExtensionFinalizedString = "files before %saccount reset"
	// HandleExtensionFormat is the formate string for a HandleExtension.
	HandleExtensionFormat = "(%s %s%s)"
	// HandleExtensionStaticTestDate is a static date used for tests (2016-03-14).
	HandleExtensionStaticTestDate = 1457913600
)

// HandleExtensionType is the type of extension.
type HandleExtensionType int

const (
	// HandleExtensionConflict means the handle conflicted as a result of a social
	// assertion resolution.
	HandleExtensionConflict HandleExtensionType = iota
	// HandleExtensionFinalized means the folder ended up with no more valid writers as
	// a result of an account reset.
	HandleExtensionFinalized
	// HandleExtensionUnknown means the type is unknown.
	HandleExtensionUnknown
)

// HandleExtensionFinalizedStringRegex is the regex identifying a finalized extension string.
var HandleExtensionFinalizedStringRegex = fmt.Sprintf(
	HandleExtensionFinalizedString, "(?:"+HandleExtensionUsernameRegex+"[\\s]+)*",
)

// HandleExtensionTypeRegex is the regular expression matching the HandleExtension string.
var HandleExtensionTypeRegex = HandleExtensionConflictString + "|" + HandleExtensionFinalizedStringRegex

// HandleExtensionFinalizedRegex is the compiled regular expression matching a finalized
// handle extension.
var HandleExtensionFinalizedRegex = regexp.MustCompile(
	fmt.Sprintf(HandleExtensionFinalizedString, "(?:("+HandleExtensionUsernameRegex+")[\\s]+)*"),
)

// String implements the fmt.Stringer interface for HandleExtensionType
func (et HandleExtensionType) String(username libkb.NormalizedUsername) string {
	switch et {
	case HandleExtensionConflict:
		return HandleExtensionConflictString
	case HandleExtensionFinalized:
		if len(username) != 0 {
			username += " "
		}
		return fmt.Sprintf(HandleExtensionFinalizedString, username)
	}
	return "<unknown extension type>"
}

// ParseHandleExtensionString parses an extension type and optional username from a string.
func ParseHandleExtensionString(s string) (HandleExtensionType, libkb.NormalizedUsername) {
	if HandleExtensionConflictString == s {
		return HandleExtensionConflict, ""
	}
	m := HandleExtensionFinalizedRegex.FindStringSubmatch(s)
	if len(m) < 2 {
		return HandleExtensionUnknown, ""
	}
	return HandleExtensionFinalized, libkb.NewNormalizedUsername(m[1])
}

// ErrHandleExtensionInvalidString is returned when a given string is not parsable as a
// valid extension suffix.
var ErrHandleExtensionInvalidString = errors.New("Invalid TLF handle extension string")

// ErrHandleExtensionInvalidNumber is returned when an invalid number is used in an
// extension definition. Handle extension numbers present in the string must be >1. Numbers
// passed to NewHandleExtension must be >0.
var ErrHandleExtensionInvalidNumber = errors.New("Invalid TLF handle extension number")

// HandleExtensionRegex is the compiled regular expression matching a valid combination
// of TLF handle extensions in string form.
var HandleExtensionRegex = regexp.MustCompile(
	fmt.Sprintf("\\"+HandleExtensionFormat,
		"("+HandleExtensionTypeRegex+")",
		"("+HandleExtensionDateRegex+")",
		"(?: #("+HandleExtensionNumberRegex+"))?\\"),
)

// HandleExtension is information which identifies a particular extension.
type HandleExtension struct {
	Date     int64                    `codec:"date"`
	Number   uint16                   `codec:"num"`
	Type     HandleExtensionType      `codec:"type"`
	Username libkb.NormalizedUsername `codec:"un,omitempty"`
	codec.UnknownFieldSetHandler
}

// String implements the fmt.Stringer interface for HandleExtension.
// Ex: "(conflicted copy 2016-05-09 #2)"
func (e HandleExtension) String() string {
	date := time.Unix(e.Date, 0).UTC().Format(HandleExtensionDateFormat)
	var num string
	if e.Number > 1 {
		num = " #"
		num += strconv.FormatUint(uint64(e.Number), 10)
	}
	return fmt.Sprintf(HandleExtensionFormat, e.Type.String(e.Username), date, num)
}

// NewHandleExtension returns a new HandleExtension struct
// populated with the date from the given time and conflict number.
func NewHandleExtension(extType HandleExtensionType, num uint16, un libkb.NormalizedUsername, now time.Time) (
	*HandleExtension, error) {
	return newHandleExtension(extType, num, un, now)
}

// NewTestHandleExtensionStaticTime returns a new HandleExtension struct populated with
// a static date for testing.
func NewTestHandleExtensionStaticTime(extType HandleExtensionType, num uint16, un libkb.NormalizedUsername) (
	*HandleExtension, error) {
	now := time.Unix(HandleExtensionStaticTestDate, 0)
	return newHandleExtension(extType, num, un, now)
}

// Helper to instantiate a HandleExtension object.
func newHandleExtension(extType HandleExtensionType, num uint16, un libkb.NormalizedUsername, now time.Time) (
	*HandleExtension, error) {
	if num == 0 {
		return nil, ErrHandleExtensionInvalidNumber
	}
	// mask out everything but the date
	date := now.UTC().Format(HandleExtensionDateFormat)
	now, err := time.Parse(HandleExtensionDateFormat, date)
	if err != nil {
		return nil, err
	}
	return &HandleExtension{
		Date:     now.UTC().Unix(),
		Number:   num,
		Type:     extType,
		Username: un,
	}, nil
}

// parseHandleExtension parses a HandleExtension array of string fields.
func parseHandleExtension(fields []string) (*HandleExtension, error) {
	if len(fields) != 4 {
		return nil, ErrHandleExtensionInvalidString
	}
	extType, un := ParseHandleExtensionString(fields[1])
	if extType == HandleExtensionUnknown {
		return nil, ErrHandleExtensionInvalidString
	}
	date, err := time.Parse(HandleExtensionDateFormat, fields[2])
	if err != nil {
		return nil, err
	}
	var num uint64 = 1
	if len(fields[3]) != 0 {
		num, err = strconv.ParseUint(fields[3], 10, 16)
		if err != nil {
			return nil, err
		}
		if num < 2 {
			return nil, ErrHandleExtensionInvalidNumber
		}
	}
	return &HandleExtension{
		Date:     date.UTC().Unix(),
		Number:   uint16(num),
		Type:     extType,
		Username: un,
	}, nil
}

// ParseHandleExtensionSuffix parses a TLF handle extension suffix string.
func ParseHandleExtensionSuffix(s string) ([]HandleExtension, error) {
	exts := HandleExtensionRegex.FindAllStringSubmatch(s, 2)
	if len(exts) < 1 || len(exts) > 2 {
		return nil, ErrHandleExtensionInvalidString
	}
	extMap := make(map[HandleExtensionType]bool)
	var extensions []HandleExtension
	for _, e := range exts {
		ext, err := parseHandleExtension(e)
		if err != nil {
			return nil, err
		}
		if extMap[ext.Type] {
			// No duplicate extension types in the same suffix.
			return nil, ErrHandleExtensionInvalidString
		}
		extMap[ext.Type] = true
		extensions = append(extensions, *ext)
	}
	return extensions, nil
}

// NewHandleExtensionSuffix creates a suffix string given a set of extensions.
func NewHandleExtensionSuffix(extensions []HandleExtension) string {
	var suffix string
	for _, extension := range extensions {
		suffix += HandleExtensionSep
		suffix += extension.String()
	}
	return suffix
}

// HandleExtensionList allows us to sort extensions by type.
type HandleExtensionList []HandleExtension

func (l HandleExtensionList) Len() int {
	return len(l)
}

func (l HandleExtensionList) Less(i, j int) bool {
	return l[i].Type < l[j].Type
}

func (l HandleExtensionList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Splat will deconstruct the list for the caller into individual extension
// pointers (or nil.)
func (l HandleExtensionList) Splat() (ci, fi *HandleExtension) {
	for _, extension := range l {
		tmp := extension
		if extension.Type == HandleExtensionConflict {
			ci = &tmp
		} else if extension.Type == HandleExtensionFinalized {
			fi = &tmp
		}
	}
	return ci, fi
}

// Suffix outputs a suffix string for this extension list.
func (l HandleExtensionList) Suffix() string {
	return NewHandleExtensionSuffix(l)
}
