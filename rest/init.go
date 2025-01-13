package rest

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
)

var (
	rdx kevlar.ReadableRedux
)

func Init() (err error) {
	rdx, err = vangogh_integration.NewReduxReader(data.AllProperties()...)
	return err
}
