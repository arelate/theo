package cli

import (
	"errors"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/redux"
)

const (
	oldTitleProperty         = "title"
	oldRequiresGamesProperty = "requires-games"
	oldBundleNameProperty    = "bundle-name"
)

func Migrate() error {

	kvRdx, err := kevlar.New(data.AbsReduxDir(), kevlar.GobExt)
	if err != nil {
		return err
	}

	if !kvRdx.Has(oldTitleProperty) {
		return nil
	}

	oldRdx, err := redux.NewReader(data.AbsReduxDir(),
		oldTitleProperty,
		oldBundleNameProperty,
		oldRequiresGamesProperty,
		data.InstallInfoProperty)

	if err != nil {
		return err
	}

	newRdx, err := redux.NewWriter(data.AbsReduxDir(),
		vangogh_integration.GogTitleProperty,
		vangogh_integration.SteamTitleProperty,
		vangogh_integration.EgsTitleProperty,
		vangogh_integration.EgsMainGameProperty,
		vangogh_integration.GogBundleNameProperty)

	if err != nil {
		return err
	}

	for id := range oldRdx.Keys(oldRequiresGamesProperty) {
		if rgs, ok := oldRdx.GetAllValues(oldRequiresGamesProperty, id); ok && len(rgs) > 0 {
			if err = newRdx.ReplaceValues(vangogh_integration.EgsMainGameProperty, id, rgs...); err != nil {
				return err
			}
		}
	}

	for id := range oldRdx.Keys(oldBundleNameProperty) {
		if bns, ok := oldRdx.GetAllValues(oldBundleNameProperty, id); ok && len(bns) > 0 {
			if err = newRdx.ReplaceValues(vangogh_integration.GogBundleNameProperty, id, bns...); err != nil {
				return err
			}
		}
	}

	for id := range oldRdx.Keys(data.InstallInfoProperty) {

		if lines, ok := oldRdx.GetAllValues(data.InstallInfoProperty, id); ok && len(lines) > 0 {

			var installInfos []InstallInfo
			installInfos, err = unmarshalInstalledInfoLines(lines...)
			if err != nil {
				return err
			}

			oldTitle, sure := oldRdx.GetLastVal(oldTitleProperty, id)
			if !sure || oldTitle == "" {
				continue
			}

			for _, ii := range installInfos {
				switch ii.Origin {
				case data.VangoghOrigin:
					if err = newRdx.ReplaceValues(vangogh_integration.GogTitleProperty, id, oldTitle); err != nil {
						return err
					}
				case data.SteamOrigin:
					if err = newRdx.ReplaceValues(vangogh_integration.SteamTitleProperty, id, oldTitle); err != nil {
						return err
					}

				case data.EpicGamesOrigin:
					if err = newRdx.ReplaceValues(vangogh_integration.EgsTitleProperty, id, oldTitle); err != nil {
						return err
					}
				default:
					return errors.New("unknown origin " + ii.Origin.String())
				}
			}
		}

	}

	// read again
	kvRdx, err = kevlar.New(data.AbsReduxDir(), kevlar.GobExt)
	if err != nil {
		return err
	}

	if err = kvRdx.Cut(oldBundleNameProperty); err != nil {
		return err
	}

	if err = kvRdx.Cut(oldRequiresGamesProperty); err != nil {
		return err
	}

	if err = kvRdx.Cut(oldTitleProperty); err != nil {
		return err
	}

	return nil
}
