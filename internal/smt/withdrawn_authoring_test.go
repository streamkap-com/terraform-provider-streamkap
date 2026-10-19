package smt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func withAuthorable(declaration string, authorable bool) string {
	value := "false"
	if authorable {
		value = "true"
	}
	return strings.Replace(declaration, `"authorable": true`, `"authorable": `+value, 1)
}

func TestCatalogEvolutionRetainsWithdrawnExactPinButDoesNotAuthorIt(t *testing.T) {
	active := withAuthorable(
		evolutionType("route", "transform", 1, 1, true, "destinations", "pattern", "string"),
		true,
	)
	withdrawn := withAuthorable(
		evolutionType("route", "transform", 2, 1, true, "destinations", "count", "integer"),
		false,
	)
	catalog := NewCatalog(evolutionCorpus(t, active, withdrawn))

	current, ok := catalog.Current("transform", "route")
	require.True(t, ok)
	require.Equal(t, int64(1), current.SchemaVersion)

	retained, ok := catalog.Lookup(CatalogPin{Role: "transform", TypeID: "route", SchemaVersion: 2, CodecVersion: 1})
	require.True(t, ok)
	require.Equal(t, int64(2), retained.SchemaVersion)
	require.Equal(t, 2, catalog.Len())
	require.Equal(t, []string{"route"}, catalog.Authorable().TypeIDs())
}

func TestParseCorpusRefusesMissingAuthorableContractField(t *testing.T) {
	missing := evolutionType("route", "transform", 1, 1, true, "destinations", "pattern", "string")
	missing = strings.Replace(missing, `  "authorable": true,`+"\n", "", 1)
	raw := `{"contract_version":1,"producer":"fixture","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","types":[` + missing + `]}`

	_, err := ParseCorpus([]byte(raw))

	require.ErrorContains(t, err, "authorable is required")
}
