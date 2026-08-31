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

var (
	msgListingTitle  = &i18n.Message{ID: "ListingTitle", Other: "Index of {{.Path}}"}
	msgColName       = &i18n.Message{ID: "ListingColName", Other: "Name"}
	msgColSize       = &i18n.Message{ID: "ListingColSize", Other: "Size"}
	msgColModified   = &i18n.Message{ID: "ListingColModified", Other: "Modified"}
	msgCrumbRoot     = &i18n.Message{ID: "CrumbRoot", Other: "root"}
	msgNotFoundTitle = &i18n.Message{ID: "NotFoundTitle", Other: "Not found"}
	msgNotFoundLead  = &i18n.Message{ID: "NotFoundLead", Other: "No file at"}
	msgNotFoundBack  = &i18n.Message{ID: "NotFoundBack", Other: "Back to /"}
	msgMethodNA      = &i18n.Message{ID: "MethodNotAllowed", Other: "method not allowed"}
	msgCLIShort      = &i18n.Message{ID: "CLIShort", Other: "Simple HTTP file server"}
	msgFlagAddr      = &i18n.Message{ID: "FlagAddr", Other: "Listen address"}
	msgFlagDir       = &i18n.Message{ID: "FlagDir", Other: "Served directory"}
	msgFlagSPA       = &i18n.Message{ID: "FlagSPA", Other: "SPA mode: never list directories"}
)

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

func localize(loc *i18n.Localizer, msg *i18n.Message, data map[string]any) string {
	s, err := loc.Localize(&i18n.LocalizeConfig{DefaultMessage: msg, TemplateData: data})
	if err != nil {
		return msg.Other
	}
	return s
}

func localizeTag(loc *i18n.Localizer) string {
	_, tag, err := loc.LocalizeWithTag(&i18n.LocalizeConfig{DefaultMessage: msgColName})
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
		Short: localize(loc, msgCLIShort, nil),
		Addr:  localize(loc, msgFlagAddr, nil),
		Dir:   localize(loc, msgFlagDir, nil),
		SPA:   localize(loc, msgFlagSPA, nil),
	}
}
