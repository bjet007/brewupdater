package brewupdater

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/bjet007/brewupdater/internal/plan"
)

const (
	upToDateText = "Everything is up to date"
	favoriteMark = "★"
	pinnedNote   = "(pinned)"
	columnGap    = 2
)

type jsonRow struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Installed string `json:"installed"`
	Latest    string `json:"latest"`
	Pinned    bool   `json:"pinned"`
	Favorite  bool   `json:"favorite"`
}

func writeList(w io.Writer, p plan.Plan) error {
	rows := p.Rows()
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, upToDateText)
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, columnGap, ' ', 0)
	for _, r := range rows {
		var cells []string
		if len(p.Favorites) > 0 {
			star := ""
			if r.Favorite {
				star = favoriteMark
			}
			cells = append(cells, star)
		}
		latest := "→ " + r.Latest
		if r.Pinned {
			latest += " " + pinnedNote
		}
		cells = append(cells, r.Name, kindName(r.Kind), r.Installed, latest)
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeJSON(w io.Writer, p plan.Plan) error {
	rows := p.Rows()
	out := make([]jsonRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, jsonRow{
			Name:      r.Name,
			Kind:      kindName(r.Kind),
			Installed: r.Installed,
			Latest:    r.Latest,
			Pinned:    r.Pinned,
			Favorite:  r.Favorite,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeWarnings(w io.Writer, p plan.Plan) {
	for _, name := range p.UnknownFavorites {
		_, _ = fmt.Fprintf(w, "Warning: favorite %q is not installed\n", name)
	}
}

func kindName(k plan.Kind) string {
	switch k {
	case plan.KindFormula:
		return "formula"
	case plan.KindCask:
		return "cask"
	}
	return ""
}
