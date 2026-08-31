package goftpd

import (
	"embed"
	"fmt"
	"io/fs"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/active.*.toml
var localeFS embed.FS

var bundle = sync.OnceValue(func() *i18n.Bundle {
	b, err := loadBundle()
	if err != nil {
		panic(err)
	}
	return b
})

func loadBundle() (*i18n.Bundle, error) {
	b := i18n.NewBundle(language.English)
	b.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	names, err := fs.Glob(localeFS, "locales/active.*.toml")
	if err != nil {
		return nil, fmt.Errorf("locale glob: %w", err)
	}
	for _, name := range names {
		if _, err := b.LoadMessageFileFS(localeFS, name); err != nil {
			return nil, fmt.Errorf("locale %s: %w", name, err)
		}
	}
	return b, nil
}

// Localizer builds a localizer from BCP 47 tags and Accept-Language values.
func Localizer(langs ...string) *i18n.Localizer {
	return i18n.NewLocalizer(bundle(), langs...)
}

func localize(loc *i18n.Localizer, id string, data map[string]any) string {
	s, err := loc.Localize(&i18n.LocalizeConfig{MessageID: id, TemplateData: data})
	if err != nil {
		return id
	}
	return s
}

func localizeTag(loc *i18n.Localizer) string {
	_, tag, err := loc.LocalizeWithTag(&i18n.LocalizeConfig{MessageID: "ListingColName"})
	if err != nil || tag == language.Und {
		return "en"
	}
	base, conf := tag.Base()
	if conf == language.No {
		return "en"
	}
	return base.String()
}

// CLICopy is cobra help text. HTTP language comes from Accept-Language,
// so the command line stays on the default catalog (English).
type CLICopy struct {
	Short, Addr, Dir, SPA string
}

// NewCLICopy localizes cobra strings in English.
func NewCLICopy() CLICopy {
	loc := Localizer()
	return CLICopy{
		Short: localize(loc, "CLIShort", nil),
		Addr:  localize(loc, "FlagAddr", nil),
		Dir:   localize(loc, "FlagDir", nil),
		SPA:   localize(loc, "FlagSPA", nil),
	}
}
