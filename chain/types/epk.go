package types

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/EpiK-Protocol/go-epik/build"
)

type EPK BigInt

func (f EPK) String() string {
	r := new(big.Rat).SetFrac(f.Int, big.NewInt(int64(build.FilecoinPrecision)))
	if r.Sign() == 0 {
		return "0 tEPK"
	}
	return strings.TrimRight(strings.TrimRight(r.FloatString(18), "0"), ".") + " tEPK"
}

func (f EPK) Format(s fmt.State, ch rune) {
	switch ch {
	case 's', 'v':
		fmt.Fprint(s, f.String())
	default:
		f.Int.Format(s, ch)
	}
}

func ParseEPK(s string) (EPK, error) {
	suffix := strings.TrimLeft(s, ".1234567890")
	s = s[:len(s)-len(suffix)]
	var attoepk bool
	if suffix != "" {
		norm := strings.ToLower(strings.TrimSpace(suffix))
		switch norm {
		case "", "tepk", "epk":
		case "attoepk", "aepk", "attotepk", "atepk":
			attoepk = true
		default:
			return EPK{}, fmt.Errorf("unrecognized suffix: %q", suffix)
		}
	}

	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return EPK{}, fmt.Errorf("failed to parse %q as a decimal number", s)
	}

	if !attoepk {
		r = r.Mul(r, big.NewRat(int64(build.FilecoinPrecision), 1))
	}

	if !r.IsInt() {
		var pref string
		if attoepk {
			pref = "atto"
		}
		return EPK{}, fmt.Errorf("invalid %stEPK value: %q", pref, s)
	}

	return EPK{r.Num()}, nil
}
