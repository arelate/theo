package rest

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

var (
	rdx redux.Readable
)

func Init() (err error) {

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err = redux.NewReader(reduxDir, data.AllProperties()...)
	return err
}
