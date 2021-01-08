package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEpkShort(t *testing.T) {
	for _, s := range []struct {
		epk    string
		expect string
	}{
		{epk: "1", expect: "1 EPK"},
		{epk: "1.1", expect: "1.1 EPK"},
		{epk: "12", expect: "12 EPK"},
		{epk: "123", expect: "123 EPK"},
		{epk: "123456", expect: "123456 EPK"},
		{epk: "123.23", expect: "123.23 EPK"},
		{epk: "123456.234", expect: "123456.234 EPK"},
		{epk: "123456.2341234", expect: "123456.234 EPK"},
		{epk: "123456.234123445", expect: "123456.234 EPK"},

		{epk: "0.1", expect: "100 mEPK"},
		{epk: "0.01", expect: "10 mEPK"},
		{epk: "0.001", expect: "1 mEPK"},

		{epk: "0.0001", expect: "100 μEPK"},
		{epk: "0.00001", expect: "10 μEPK"},
		{epk: "0.000001", expect: "1 μEPK"},

		{epk: "0.0000001", expect: "100 nEPK"},
		{epk: "0.00000001", expect: "10 nEPK"},
		{epk: "0.000000001", expect: "1 nEPK"},

		{epk: "0.0000000001", expect: "100 pEPK"},
		{epk: "0.00000000001", expect: "10 pEPK"},
		{epk: "0.000000000001", expect: "1 pEPK"},

		{epk: "0.0000000000001", expect: "100 fEPK"},
		{epk: "0.00000000000001", expect: "10 fEPK"},
		{epk: "0.000000000000001", expect: "1 fEPK"},

		{epk: "0.0000000000000001", expect: "100 aEPK"},
		{epk: "0.00000000000000001", expect: "10 aEPK"},
		{epk: "0.000000000000000001", expect: "1 aEPK"},

		{epk: "0.0000012", expect: "1.2 μEPK"},
		{epk: "0.00000123", expect: "1.23 μEPK"},
		{epk: "0.000001234", expect: "1.234 μEPK"},
		{epk: "0.0000012344", expect: "1.234 μEPK"},
		{epk: "0.00000123444", expect: "1.234 μEPK"},

		{epk: "0.0002212", expect: "221.2 μEPK"},
		{epk: "0.00022123", expect: "221.23 μEPK"},
		{epk: "0.000221234", expect: "221.234 μEPK"},
		{epk: "0.0002212344", expect: "221.234 μEPK"},
		{epk: "0.00022123444", expect: "221.234 μEPK"},
	} {
		t.Run(s.epk, func(t *testing.T) {
			f, err := ParseEPK(s.epk)
			require.NoError(t, err)
			require.Equal(t, s.expect, f.Short())
		})
	}
}
