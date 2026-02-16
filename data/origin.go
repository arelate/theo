package data

import (
	"errors"
	"maps"
	"slices"
)

type Origin int

const (
	UnknownOrigin Origin = iota
	VangoghGogOrigin
	SteamOrigin
)

var originStrings = map[Origin]string{
	UnknownOrigin:    "unknown",
	VangoghGogOrigin: "vangogh-gog",
	SteamOrigin:      "Steam",
}

func (o Origin) String() string {
	if ostr, ok := originStrings[o]; ok {
		return ostr
	}
	return originStrings[UnknownOrigin]
}

func (o Origin) ErrUnsupportedOrigin() error {
	return errors.New("unsupported origin: " + o.String())
}

func AllOrigins() []string {
	return slices.Collect(maps.Values(originStrings))
}
