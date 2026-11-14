package cli

import (
	"encoding/json"
	"fmt"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

func runLiteralContinuation(cmd *cobra.Command, root, cursor string, jsonOut bool) error {
	d, err := db.OpenIndexReadOnly(root)
	if err != nil {
		return err
	}
	defer d.Close()
	page, err := search.LiteralMatches(d, search.Filters{}, cursor)
	if err != nil {
		return err
	}
	out := search.Output{Query: page.Query, Mode: "literal-continuation", Results: []search.Result{}, Literal: page}
	if jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	text, _ := formatDigest(&out)
	fmt.Fprint(cmd.OutOrStdout(), text)
	return nil
}
