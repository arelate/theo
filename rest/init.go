package rest

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
)

var (
	rdx kevlar.ReadableRedux
)

func Init() (err error) {
	rdx, err = vangogh_local_data.NewReduxReader(data.AllProperties()...)
	return err
}
