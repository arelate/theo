package data

import (
	"errors"
	"maps"
	"slices"
)

type Origin int

const (
	UnknownOrigin Origin = iota
	VangoghOrigin
	SteamOrigin
	EpicGamesOrigin
	GogOrigin
)

var originStrings = map[Origin]string{
	UnknownOrigin:   "unknown",
	VangoghOrigin:   "vangogh",
	SteamOrigin:     "Steam",
	EpicGamesOrigin: "EGS",
	GogOrigin:       "GOG",
}

func (o Origin) String() string {
	if ostr, ok := originStrings[o]; ok {
		return ostr
	}
	return originStrings[UnknownOrigin]
}

func ParseOrigin(originStr string) Origin {
	for org, str := range originStrings {
		if str == originStr {
			return org
		}
	}
	return UnknownOrigin
}

func (o Origin) ErrUnsupportedOrigin() error {
	return errors.New("unsupported origin: " + o.String())
}

func AllOrigins() []string {
	return slices.Collect(maps.Values(originStrings))
}
