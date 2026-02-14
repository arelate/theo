package data

import "errors"

type Origin int

const (
	UnknownOrigin Origin = iota
	VangoghOrigin
	SteamOrigin
)

var originStrings = map[Origin]string{
	UnknownOrigin: "unknown",
	VangoghOrigin: "vangogh",
	SteamOrigin:   "Steam",
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
